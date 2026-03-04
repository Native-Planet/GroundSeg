package broadcast

import (
	"time"

	"go.uber.org/zap"
	"groundseg/auth"
	"groundseg/leak"
	"groundseg/structs"
)

type authBroadcastEnvelope struct {
	Type      string                `json:"type"`
	AuthLevel string                `json:"auth_level"`
	Payload   structs.AuthBroadcast `json:"payload"`
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	return broadcastToClientsWithRuntime(DefaultBroadcastStateRuntime())
}

func broadcastToClientsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		runtime = DefaultBroadcastStateRuntime()
	}
	bState := runtime.GetState()
	sent := false
	for attempt := 0; attempt < 5; attempt++ {
		select {
		case leak.LeakChan <- bState:
			sent = true
			attempt = 5
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !sent {
		zap.L().Warn("Dropping broadcast update: leak broadcaster is congested")
	}

	cm := auth.GetClientManager()
	if cm == nil || !cm.HasAuthSession() {
		return nil
	}
	authJson, err := GetStateJson(bState)
	if err != nil {
		return err
	}
	if err := cm.BroadcastAuth(authJson); err != nil {
		return err
	}
	return nil
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	cm := auth.GetClientManager()
	if cm == nil {
		return nil
	}
	if err := cm.BroadcastUnauth(input); err != nil {
		return err
	}
	return nil
}
