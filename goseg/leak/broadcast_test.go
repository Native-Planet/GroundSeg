package leak

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"groundseg/structs"
)

func resetLeakStateForTest(t *testing.T) {
	t.Helper()
	originalBytes := BytesChan
	lickMu.Lock()
	originalStatuses := lickStatuses
	lickMu.Unlock()

	BytesChan = map[string]chan string{}
	lickMu.Lock()
	lickStatuses = map[string]LickStatus{}
	lickMu.Unlock()

	t.Cleanup(func() {
		BytesChan = originalBytes
		lickMu.Lock()
		lickStatuses = originalStatuses
		lickMu.Unlock()
	})
}

func setLickStatusesForTest(statuses map[string]LickStatus) {
	lickMu.Lock()
	lickStatuses = statuses
	lickMu.Unlock()
}

func mustReceiveString(t *testing.T, ch <-chan string, name string) string {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return ""
	}
}

func TestUpdateBroadcastReturnsOriginalWhenUnchanged(t *testing.T) {
	resetLeakStateForTest(t)
	BytesChan["~zod"] = make(chan string, 1)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: true}})

	oldBroadcast := structs.AuthBroadcast{
		Type:      "structure",
		AuthLevel: "authorized",
		Urbits: map[string]structs.Urbit{
			"~zod": {},
		},
	}

	updated, err := updateBroadcast(oldBroadcast, oldBroadcast)
	if err != nil {
		t.Fatalf("updateBroadcast returned error: %v", err)
	}
	if !reflect.DeepEqual(updated, oldBroadcast) {
		t.Fatalf("expected unchanged broadcast, got %+v", updated)
	}

	select {
	case msg := <-BytesChan["~zod"]:
		t.Fatalf("expected no broadcast to be sent, got %q", msg)
	default:
	}
}

func TestUpdateBroadcastSendsFullAndScopedPayloads(t *testing.T) {
	resetLeakStateForTest(t)
	BytesChan["~zod"] = make(chan string, 1)
	BytesChan["~bus"] = make(chan string, 1)
	BytesChan["~nec"] = make(chan string, 1)
	setLickStatusesForTest(map[string]LickStatus{
		"~zod": {Auth: true},
		"~bus": {Auth: false},
		"~nec": {Auth: false},
	})

	newBroadcast := structs.AuthBroadcast{
		Type:      "structure",
		AuthLevel: "authorized",
		Urbits: map[string]structs.Urbit{
			"~zod": {},
			"~bus": {},
		},
	}
	newBroadcast.Profile.Startram.Info.Registered = true
	newBroadcast.Profile.Startram.Info.Running = true

	updated, err := updateBroadcast(structs.AuthBroadcast{}, newBroadcast)
	if err != nil {
		t.Fatalf("updateBroadcast returned error: %v", err)
	}
	if !reflect.DeepEqual(updated, newBroadcast) {
		t.Fatalf("unexpected updated payload: %+v", updated)
	}

	var full structs.AuthBroadcast
	if err := json.Unmarshal([]byte(mustReceiveString(t, BytesChan["~zod"], "authorized broadcast")), &full); err != nil {
		t.Fatalf("failed to unmarshal authorized broadcast: %v", err)
	}
	if full.AuthLevel != "authorized" {
		t.Fatalf("unexpected authorized auth level: %q", full.AuthLevel)
	}
	if len(full.Urbits) != 2 {
		t.Fatalf("expected full urbit map for authorized ship, got %d entries", len(full.Urbits))
	}

	var scoped structs.AuthBroadcast
	if err := json.Unmarshal([]byte(mustReceiveString(t, BytesChan["~bus"], "scoped broadcast")), &scoped); err != nil {
		t.Fatalf("failed to unmarshal scoped broadcast: %v", err)
	}
	if scoped.Type != "structure" {
		t.Fatalf("unexpected scoped type: %q", scoped.Type)
	}
	if scoped.AuthLevel != "~bus" {
		t.Fatalf("unexpected scoped auth level: %q", scoped.AuthLevel)
	}
	if len(scoped.Urbits) != 1 {
		t.Fatalf("expected exactly one urbit in scoped broadcast, got %d", len(scoped.Urbits))
	}
	if _, exists := scoped.Urbits["~bus"]; !exists {
		t.Fatalf("expected scoped broadcast to contain ~bus, got %+v", scoped.Urbits)
	}
	if !scoped.Profile.Startram.Info.Registered || !scoped.Profile.Startram.Info.Running {
		t.Fatalf("expected startram status to be preserved, got %+v", scoped.Profile.Startram.Info)
	}

	select {
	case msg := <-BytesChan["~nec"]:
		t.Fatalf("expected no scoped payload for missing urbit, got %q", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

type failingConn struct{}

func (f *failingConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (f *failingConn) Write(_ []byte) (int, error)        { return 0, errors.New("write failed") }
func (f *failingConn) Close() error                       { return nil }
func (f *failingConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (f *failingConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (f *failingConn) SetDeadline(_ time.Time) error      { return nil }
func (f *failingConn) SetReadDeadline(_ time.Time) error  { return nil }
func (f *failingConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestSendBroadcastNilConnectionIsNoop(t *testing.T) {
	conn, err := sendBroadcast(nil, `{"ok":true}`)
	if err != nil {
		t.Fatalf("sendBroadcast(nil) returned error: %v", err)
	}
	if conn != nil {
		t.Fatalf("expected nil conn return, got %#v", conn)
	}
}

func TestSendBroadcastWritesJammedPayload(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	payloadRead := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		if err := server.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			readErr <- err
			return
		}
		n, err := server.Read(buf)
		if err != nil {
			readErr <- err
			return
		}
		payloadRead <- append([]byte(nil), buf[:n]...)
	}()

	returnedConn, err := sendBroadcast(client, `{"hello":"world"}`)
	if err != nil {
		t.Fatalf("sendBroadcast returned error: %v", err)
	}
	if returnedConn != client {
		t.Fatalf("expected sendBroadcast to return original conn")
	}

	select {
	case err := <-readErr:
		t.Fatalf("failed to read jammed payload: %v", err)
	case payload := <-payloadRead:
		if len(payload) == 0 {
			t.Fatal("expected non-empty jammed payload")
		}
		if payload[0] != 0 {
			t.Fatalf("expected jammed payload to start with version byte 0, got %d", payload[0])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for jammed payload")
	}
}

func TestSendBroadcastReturnsErrorWhenConnectionWriteFails(t *testing.T) {
	conn, err := sendBroadcast(&failingConn{}, "payload")
	if err == nil {
		t.Fatal("expected write error")
	}
	if conn != nil {
		t.Fatalf("expected nil conn on write failure, got %#v", conn)
	}
}

func TestListenerCleansStateWhenConnectionCloses(t *testing.T) {
	resetLeakStateForTest(t)
	server, client := net.Pipe()
	defer server.Close()

	done := make(chan struct{})
	go func() {
		listener("~zod", client, LickStatus{Auth: true})
		close(done)
	}()

	var shipChan chan string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ch, ok := BytesChan["~zod"]; ok {
			shipChan = ch
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if shipChan == nil {
		t.Fatal("listener did not register ship channel")
	}

	readDone := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		if err := server.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			readErr <- err
			return
		}
		n, err := server.Read(buf)
		if err != nil {
			readErr <- err
			return
		}
		readDone <- append([]byte(nil), buf[:n]...)
	}()

	shipChan <- `{"kind":"test"}`
	select {
	case err := <-readErr:
		t.Fatalf("failed to read listener broadcast: %v", err)
	case payload := <-readDone:
		if len(payload) == 0 {
			t.Fatal("expected listener to write payload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for listener broadcast")
	}

	if err := server.Close(); err != nil {
		t.Fatalf("failed to close server side of pipe: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit after connection close")
	}

	if _, exists := BytesChan["~zod"]; exists {
		t.Fatal("expected listener cleanup to remove BytesChan entry")
	}
	if _, exists := GetLickStatuses()["~zod"]; exists {
		t.Fatal("expected listener cleanup to remove lick status")
	}
}
