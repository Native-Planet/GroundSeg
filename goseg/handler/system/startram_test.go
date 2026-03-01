package system

import (
	"encoding/json"
	"testing"
	"time"

	"groundseg/structs"
)

type startramDispatchCall struct {
	action   string
	key      string
	region   string
	reset    bool
	endpoint string
	remind   bool
	password string
}

func buildStartramMessage(t *testing.T, action structs.WsStartramAction) []byte {
	t.Helper()
	payload := structs.WsStartramPayload{
		Payload: action,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal startram payload: %v", err)
	}
	return data
}

func resetStartramActionHandlersForTest() func() {
	originalServices := startramServicesActionHandler
	originalRegions := startramRegionsActionHandler
	originalRegister := startramRegisterActionHandler
	originalToggle := startramToggleActionHandler
	originalRestart := startramRestartActionHandler
	originalCancel := startramCancelActionHandler
	originalEndpoint := startramEndpointActionHandler
	originalReminder := startramReminderActionHandler
	originalSetBackup := startramSetBackupPWHandler

	startramServicesActionHandler = func() error { return nil }
	startramRegionsActionHandler = func() error { return nil }
	startramRegisterActionHandler = func(string, string) error { return nil }
	startramToggleActionHandler = func() error { return nil }
	startramRestartActionHandler = func() error { return nil }
	startramCancelActionHandler = func(string, bool) error { return nil }
	startramEndpointActionHandler = func(string) error { return nil }
	startramReminderActionHandler = func(bool) error { return nil }
	startramSetBackupPWHandler = func(string) error { return nil }

	return func() {
		startramServicesActionHandler = originalServices
		startramRegionsActionHandler = originalRegions
		startramRegisterActionHandler = originalRegister
		startramToggleActionHandler = originalToggle
		startramRestartActionHandler = originalRestart
		startramCancelActionHandler = originalCancel
		startramEndpointActionHandler = originalEndpoint
		startramReminderActionHandler = originalReminder
		startramSetBackupPWHandler = originalSetBackup
	}
}

func waitForStartramCall(t *testing.T, calls <-chan startramDispatchCall) startramDispatchCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for startram dispatch call")
		return startramDispatchCall{}
	}
}

func TestStartramHandlerDispatchesActions(t *testing.T) {
	testCases := []struct {
		name    string
		payload structs.WsStartramAction
		setup   func(calls chan<- startramDispatchCall)
		want    startramDispatchCall
	}{
		{
			name:    "services",
			payload: structs.WsStartramAction{Action: "services"},
			setup: func(calls chan<- startramDispatchCall) {
				startramServicesActionHandler = func() error {
					calls <- startramDispatchCall{action: "services"}
					return nil
				}
			},
			want: startramDispatchCall{action: "services"},
		},
		{
			name:    "regions",
			payload: structs.WsStartramAction{Action: "regions"},
			setup: func(calls chan<- startramDispatchCall) {
				startramRegionsActionHandler = func() error {
					calls <- startramDispatchCall{action: "regions"}
					return nil
				}
			},
			want: startramDispatchCall{action: "regions"},
		},
		{
			name:    "register",
			payload: structs.WsStartramAction{Action: "register", Key: "reg-key", Region: "us-east"},
			setup: func(calls chan<- startramDispatchCall) {
				startramRegisterActionHandler = func(key, region string) error {
					calls <- startramDispatchCall{action: "register", key: key, region: region}
					return nil
				}
			},
			want: startramDispatchCall{action: "register", key: "reg-key", region: "us-east"},
		},
		{
			name:    "toggle",
			payload: structs.WsStartramAction{Action: "toggle"},
			setup: func(calls chan<- startramDispatchCall) {
				startramToggleActionHandler = func() error {
					calls <- startramDispatchCall{action: "toggle"}
					return nil
				}
			},
			want: startramDispatchCall{action: "toggle"},
		},
		{
			name:    "restart",
			payload: structs.WsStartramAction{Action: "restart"},
			setup: func(calls chan<- startramDispatchCall) {
				startramRestartActionHandler = func() error {
					calls <- startramDispatchCall{action: "restart"}
					return nil
				}
			},
			want: startramDispatchCall{action: "restart"},
		},
		{
			name:    "cancel",
			payload: structs.WsStartramAction{Action: "cancel", Key: "cancel-key", Reset: true},
			setup: func(calls chan<- startramDispatchCall) {
				startramCancelActionHandler = func(key string, reset bool) error {
					calls <- startramDispatchCall{action: "cancel", key: key, reset: reset}
					return nil
				}
			},
			want: startramDispatchCall{action: "cancel", key: "cancel-key", reset: true},
		},
		{
			name:    "endpoint",
			payload: structs.WsStartramAction{Action: "endpoint", Endpoint: "api.example.com"},
			setup: func(calls chan<- startramDispatchCall) {
				startramEndpointActionHandler = func(endpoint string) error {
					calls <- startramDispatchCall{action: "endpoint", endpoint: endpoint}
					return nil
				}
			},
			want: startramDispatchCall{action: "endpoint", endpoint: "api.example.com"},
		},
		{
			name:    "reminder",
			payload: structs.WsStartramAction{Action: "reminder", Remind: true},
			setup: func(calls chan<- startramDispatchCall) {
				startramReminderActionHandler = func(remind bool) error {
					calls <- startramDispatchCall{action: "reminder", remind: remind}
					return nil
				}
			},
			want: startramDispatchCall{action: "reminder", remind: true},
		},
		{
			name:    "set-backup-password",
			payload: structs.WsStartramAction{Action: "set-backup-password", Password: "pw123"},
			setup: func(calls chan<- startramDispatchCall) {
				startramSetBackupPWHandler = func(password string) error {
					calls <- startramDispatchCall{action: "set-backup-password", password: password}
					return nil
				}
			},
			want: startramDispatchCall{action: "set-backup-password", password: "pw123"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			restore := resetStartramActionHandlersForTest()
			t.Cleanup(restore)

			calls := make(chan startramDispatchCall, 1)
			tc.setup(calls)

			if err := StartramHandler(buildStartramMessage(t, tc.payload)); err != nil {
				t.Fatalf("StartramHandler returned error: %v", err)
			}

			got := waitForStartramCall(t, calls)
			if got != tc.want {
				t.Fatalf("unexpected dispatch call: want %+v got %+v", tc.want, got)
			}
		})
	}
}

func TestStartramHandlerRejectsInvalidPayload(t *testing.T) {
	if err := StartramHandler([]byte("{bad-json")); err == nil {
		t.Fatal("expected json unmarshal error")
	}
}

func TestStartramHandlerRejectsUnknownAction(t *testing.T) {
	err := StartramHandler(buildStartramMessage(t, structs.WsStartramAction{Action: "unknown"}))
	if err == nil {
		t.Fatal("expected unknown action error")
	}
}

func TestAppendOrchestrationErrorWrapsContext(t *testing.T) {
	var errs []error
	appendOrchestrationError(&errs, "restart wireguard", nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors when input error is nil, got %d", len(errs))
	}

	appendOrchestrationError(&errs, "restart wireguard", errSentinel)
	if len(errs) != 1 {
		t.Fatalf("expected one wrapped error, got %d", len(errs))
	}
	if errs[0].Error() != "restart wireguard: sentinel" {
		t.Fatalf("unexpected wrapped error: %v", errs[0])
	}
}

var errSentinel = &startramTestError{message: "sentinel"}

type startramTestError struct {
	message string
}

func (err *startramTestError) Error() string {
	return err.message
}
