package broadcast

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/config"
	"groundseg/setup"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const sseHeartbeatInterval = 25 * time.Second

type streamAuthRequest struct {
	Token structs.WsTokenStruct `json:"token"`
}

type streamEvent struct {
	ID   uint64
	Name string
	Data []byte
}

type streamClient struct {
	TokenID string
	Authed  bool
	Ch      chan streamEvent
}

type streamHub struct {
	mu            sync.RWMutex
	authClients   map[*streamClient]struct{}
	unauthClients map[*streamClient]struct{}
	seq           atomic.Uint64
}

var eventStreams = newStreamHub()

func newStreamHub() *streamHub {
	return &streamHub{
		authClients:   make(map[*streamClient]struct{}),
		unauthClients: make(map[*streamClient]struct{}),
	}
}

func (h *streamHub) register(tokenID string, authed bool) *streamClient {
	client := &streamClient{
		TokenID: tokenID,
		Authed:  authed,
		Ch:      make(chan streamEvent, 4),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if authed {
		h.authClients[client] = struct{}{}
	} else {
		h.unauthClients[client] = struct{}{}
	}
	return client
}

func (h *streamHub) unregister(client *streamClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.authClients, client)
	delete(h.unauthClients, client)
}

func (h *streamHub) hasAuthSession() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.authClients) > 0
}

func (h *streamHub) hasUnauthSession() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.unauthClients) > 0
}

func (h *streamHub) broadcast(authed bool, name string, data []byte) {
	event := streamEvent{
		ID:   h.seq.Add(1),
		Name: name,
		Data: data,
	}
	stale := []*streamClient{}
	h.mu.RLock()
	clients := h.unauthClients
	if authed {
		clients = h.authClients
	}
	for client := range clients {
		if authed && !auth.TokenIdAuthed(auth.ClientManager, client.TokenID) {
			stale = append(stale, client)
			continue
		}
		client.send(event)
	}
	h.mu.RUnlock()
	if len(stale) < 1 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, client := range stale {
		delete(h.authClients, client)
	}
}

func (c *streamClient) send(event streamEvent) {
	select {
	case c.Ch <- event:
	default:
		select {
		case <-c.Ch:
		default:
		}
		select {
		case c.Ch <- event:
		default:
		}
	}
}

// HasEventAuthSession reports whether any authorized SSE clients are connected.
func HasEventAuthSession() bool {
	return eventStreams.hasAuthSession()
}

func hasEventUnauthSession() bool {
	return eventStreams.hasUnauthSession()
}

func broadcastAuthEvent(data []byte) {
	eventStreams.broadcast(true, streamEventName(data), data)
}

func broadcastUnauthEvent(data []byte) {
	eventStreams.broadcast(false, streamEventName(data), data)
}

// EventsHandler serves the high-frequency state channel as SSE.
func EventsHandler(w http.ResponseWriter, r *http.Request) {
	setStreamCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token, err := streamTokenFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, valid, authed := auth.CheckStreamToken(token, r)
	if !valid {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	client := eventStreams.register(token.ID, authed)
	defer eventStreams.unregister(client)

	if err := writeStreamComment(w, "connected"); err != nil {
		return
	}
	flusher.Flush()

	for _, event := range initialStreamEvents(authed) {
		if err := writeStreamEvent(w, event); err != nil {
			return
		}
		flusher.Flush()
	}

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client.Ch:
			if err := writeStreamEvent(w, event); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if err := writeStreamComment(w, "ping"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func setStreamCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Vary", "Origin")
}

func streamTokenFromRequest(r *http.Request) (structs.WsTokenStruct, error) {
	defer r.Body.Close()
	var req streamAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req.Token, fmt.Errorf("invalid token payload")
	}
	if req.Token.ID == "" || req.Token.Token == "" {
		return req.Token, fmt.Errorf("missing token")
	}
	return req.Token, nil
}

func initialStreamEvents(authed bool) []streamEvent {
	events := []streamEvent{}
	if system.IsC2CMode() {
		if data, err := json.Marshal(structs.C2CBroadcast{
			Type:  "c2c",
			SSIDS: system.C2CStoredSSIDs,
		}); err == nil {
			events = append(events, streamEvent{Name: "c2c", Data: data})
		}
		return events
	}

	conf := config.Conf()
	if conf.Setup != "complete" {
		if data, err := json.Marshal(structs.SetupBroadcast{
			Type:      "structure",
			AuthLevel: "setup",
			Stage:     conf.Setup,
			Page:      setup.Stages[conf.Setup],
			Regions:   startram.Regions,
		}); err == nil {
			events = append(events, streamEvent{Name: "structure", Data: data})
		}
		return events
	}

	if authed {
		if data, err := GetStateJson(GetState()); err == nil {
			events = append(events, streamEvent{Name: "structure", Data: data})
		}
		return events
	}

	var unauth structs.UnauthBroadcast
	unauth.Type = "structure"
	unauth.AuthLevel = "unauthorized"
	if data, err := json.Marshal(unauth); err == nil {
		events = append(events, streamEvent{Name: "structure", Data: data})
	}
	return events
}

func streamEventName(data []byte) string {
	var typed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typed); err != nil || typed.Type == "" {
		return "message"
	}
	return typed.Type
}

func writeStreamEvent(w io.Writer, event streamEvent) error {
	if event.ID > 0 {
		if _, err := fmt.Fprintf(w, "id: %d\n", event.ID); err != nil {
			return err
		}
	}
	if event.Name != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Name); err != nil {
			return err
		}
	}
	for line := range strings.SplitSeq(string(event.Data), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func writeStreamComment(w io.Writer, text string) error {
	_, err := fmt.Fprintf(w, ": %s\n\n", text)
	return err
}
