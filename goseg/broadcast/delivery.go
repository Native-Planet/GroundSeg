package broadcast

import (
	"errors"
	"fmt"
	"time"

	"groundseg/leak"
	"groundseg/session"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var ErrBroadcastLeakBackpressure = errors.New("broadcast leak channel is full")

type broadcastDeliveryRuntime struct {
	publishLeakFn       func(structs.AuthBroadcast) error
	getClientManagerFn  func() *structs.ClientManager
	hasAuthorizedFn     func(*structs.ClientManager) bool
	broadcastAuthorized func(*structs.ClientManager, []byte) error
	broadcastUnauthFn   func(*structs.ClientManager, []byte) error
}

func defaultBroadcastDeliveryRuntime() broadcastDeliveryRuntime {
	return broadcastDeliveryRuntime{
		publishLeakFn: publishLeakBroadcast,
		getClientManagerFn: func() *structs.ClientManager {
			return session.GetClientManager()
		},
		hasAuthorizedFn: func(cm *structs.ClientManager) bool {
			return cm != nil && cm.HasAuthSession()
		},
		broadcastAuthorized: func(cm *structs.ClientManager, payload []byte) error {
			if cm == nil {
				return nil
			}
			return cm.BroadcastAuth(payload)
		},
		broadcastUnauthFn: func(cm *structs.ClientManager, payload []byte) error {
			if cm == nil {
				return nil
			}
			return cm.BroadcastUnauth(payload)
		},
	}
}

func publishLeakBroadcast(state structs.AuthBroadcast) error {
	const maxLeakSendAttempts = 5
	for attempt := 0; attempt < maxLeakSendAttempts; attempt++ {
		select {
		case leak.LeakChan <- state:
			return nil
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	zap.L().Warn("Dropping broadcast update: leak broadcaster is congested")
	return fmt.Errorf("%w after %d attempts", ErrBroadcastLeakBackpressure, maxLeakSendAttempts)
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	return broadcastToClientsWithRuntime(DefaultBroadcastStateRuntime())
}

func broadcastToClientsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("broadcast to clients: %w", ErrBroadcastRuntimeRequired)
	}
	delivery := runtime.delivery
	if delivery.publishLeakFn == nil {
		delivery.publishLeakFn = publishLeakBroadcast
	}
	if delivery.getClientManagerFn == nil {
		delivery.getClientManagerFn = func() *structs.ClientManager { return session.GetClientManager() }
	}
	if delivery.hasAuthorizedFn == nil {
		delivery.hasAuthorizedFn = func(cm *structs.ClientManager) bool { return cm != nil && cm.HasAuthSession() }
	}
	if delivery.broadcastAuthorized == nil {
		delivery.broadcastAuthorized = func(cm *structs.ClientManager, payload []byte) error {
			if cm == nil {
				return nil
			}
			return cm.BroadcastAuth(payload)
		}
	}
	bState := runtime.GetState()
	if err := delivery.publishLeakFn(bState); err != nil {
		return err
	}

	cm := delivery.getClientManagerFn()
	if !delivery.hasAuthorizedFn(cm) {
		return nil
	}
	authJson, err := GetStateJson(bState, transition.BroadcastAuthLevelAuthorized)
	if err != nil {
		return err
	}
	if err := delivery.broadcastAuthorized(cm, authJson); err != nil {
		return err
	}
	return nil
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	runtime := DefaultBroadcastStateRuntime()
	if runtime == nil {
		return fmt.Errorf("unauth broadcast: %w", ErrBroadcastRuntimeRequired)
	}
	delivery := runtime.delivery
	if delivery.getClientManagerFn == nil {
		delivery.getClientManagerFn = func() *structs.ClientManager { return session.GetClientManager() }
	}
	if delivery.broadcastUnauthFn == nil {
		delivery.broadcastUnauthFn = func(cm *structs.ClientManager, payload []byte) error {
			if cm == nil {
				return nil
			}
			return cm.BroadcastUnauth(payload)
		}
	}
	if err := delivery.broadcastUnauthFn(delivery.getClientManagerFn(), input); err != nil {
		return err
	}
	return nil
}
