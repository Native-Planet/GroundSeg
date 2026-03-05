package system

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/protocol/actions"
)

func TestProcessMessageRoutesSupportedAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	adapter := newCaptiveTransportAdapter(defaultC2CServiceDeps())
	var callCount int
	adapter.processC2CMessage = func(msg []byte) error {
		callCount++
		var payload actions.C2CPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			t.Fatalf("decode message payload: %v", err)
		}
		if payload.Type != actions.C2CPayloadType {
			t.Fatalf("unexpected payload type: %q", payload.Type)
		}
		if payload.Payload.Action != string(actions.ActionC2CConnect) {
			t.Fatalf("unexpected payload action: %q", payload.Payload.Action)
		}
		if payload.Payload.SSID != "HomeWiFi" || payload.Payload.Password != "secret" {
			t.Fatalf("unexpected payload credentials: %+v", payload.Payload)
		}
		return nil
	}

	msg, err := actions.MarshalC2CPayload(actions.ActionC2CConnect, "HomeWiFi", "secret")
	if err != nil {
		t.Fatalf("marshal c2c payload: %v", err)
	}
	if err := adapter.processMessage(msg); err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected c2c message processor to run once; got %d", callCount)
	}
}

func TestProcessMessageRejectsUnsupportedAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	adapter := newCaptiveTransportAdapter(defaultC2CServiceDeps())

	msg, err := actions.MarshalC2CPayload(actions.Action("unsupported"), "HomeWiFi", "secret")
	if err != nil {
		t.Fatalf("marshal unsupported c2c payload: %v", err)
	}
	processErr := adapter.processMessage(msg)
	if processErr == nil {
		t.Fatal("expected unsupported action error")
	}
	if _, ok := processErr.(actions.UnsupportedActionError); !ok {
		t.Fatalf("expected unsupported action error, got %T: %v", processErr, processErr)
	}
}

func TestProcessMessageRejectsWrongEnvelopeType(t *testing.T) {
	resetWifiSeamsForTest(t)
	adapter := newCaptiveTransportAdapter(defaultC2CServiceDeps())

	msg, err := json.Marshal(actions.C2CPayload{
		Type: "auth",
		Payload: actions.C2CPayloadBody{
			Action:   string(actions.ActionC2CConnect),
			SSID:     "HomeWiFi",
			Password: "secret",
		},
	})
	if err != nil {
		t.Fatalf("marshal wrong-type payload: %v", err)
	}
	processErr := adapter.processMessage(msg)
	if processErr == nil {
		t.Fatal("expected wrong envelope type error")
	}
}

func TestProcessC2CMessageForAdapter(t *testing.T) {
	resetWifiSeamsForTest(t)
	connectCalled := false
	restartCalled := false
	deps := c2cServiceDeps{
		connectToWiFi: func(ssid, password string) error {
			connectCalled = true
			if ssid != "HomeWiFi" || password != "secret" {
				t.Fatalf("unexpected credentials: %s / %s", ssid, password)
			}
			return nil
		},
		restartGroundSeg: func() error {
			restartCalled = true
			return nil
		},
	}
	msg, err := actions.MarshalC2CPayload(actions.ActionC2CConnect, "HomeWiFi", "secret")
	if err != nil {
		t.Fatalf("marshal c2c payload: %v", err)
	}
	if err := processC2CMessageForAdapterWithDeps(msg, deps); err != nil {
		t.Fatalf("processC2CMessageForAdapter failed: %v", err)
	}
	if !connectCalled || !restartCalled {
		t.Fatalf("expected connect and restart to be called")
	}
}

func TestC2CServiceExecutesConnectAndRestart(t *testing.T) {
	resetWifiSeamsForTest(t)
	connectCalled := false
	restartCalled := false
	deps := c2cServiceDeps{
		connectToWiFi: func(ssid, password string) error {
			connectCalled = true
			if ssid != "HomeWiFi" || password != "secret" {
				t.Fatalf("unexpected credentials: %s / %s", ssid, password)
			}
			return nil
		},
		restartGroundSeg: func() error {
			restartCalled = true
			return nil
		},
	}
	service, err := newC2CServiceForAdapterWithDeps(deps)
	if err != nil {
		t.Fatalf("newC2CServiceForAdapterWithDeps failed: %v", err)
	}

	if err := service.Execute(c2cActionConnect, "HomeWiFi", "secret"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !connectCalled || !restartCalled {
		t.Fatalf("expected connect and restart to be called")
	}
}

func TestNewC2CServiceForAdapterWithDepsRejectsNilDependencies(t *testing.T) {
	resetWifiSeamsForTest(t)

	_, err := newC2CServiceForAdapterWithDeps(c2cServiceDeps{
		restartGroundSeg: func() error { return nil },
	})
	if err == nil {
		t.Fatal("expected constructor to fail when connect callback is nil")
	}

	_, err = newC2CServiceForAdapterWithDeps(c2cServiceDeps{
		connectToWiFi: func(_, _ string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected constructor to fail when restart callback is nil")
	}
}

func TestProcessC2CMessageForAdapterReportsMissingServiceOnConstructorFailure(t *testing.T) {
	resetWifiSeamsForTest(t)

	msg, err := actions.MarshalC2CPayload(actions.ActionC2CConnect, "HomeWiFi", "secret")
	if err != nil {
		t.Fatalf("marshal c2c payload: %v", err)
	}
	deps := c2cServiceDeps{
		connectToWiFi: nil,
	}

	err = processC2CMessageForAdapterWithDeps(msg, deps)
	if err == nil {
		t.Fatal("expected processC2CMessageForAdapter to fail when dependencies are nil")
	}
	if !strings.Contains(err.Error(), "c2c service is required") {
		t.Fatalf("expected service requirement error, got %v", err)
	}
}

type wifiRadioConnectErrorService struct {
	connectedDevice string
	connectErr      error
}

func (s wifiRadioConnectErrorService) PrimaryDevice() (string, error) {
	return s.connectedDevice, nil
}

func (s wifiRadioConnectErrorService) RefreshInfo(_ string) {}

func (s wifiRadioConnectErrorService) Enable() error {
	return nil
}

func (s wifiRadioConnectErrorService) SetLinkUp(_ string) error {
	return nil
}

func (s wifiRadioConnectErrorService) Connect(_, _ string) error {
	return s.connectErr
}

func (s wifiRadioConnectErrorService) ListSSIDs(_ string) ([]string, error) {
	return nil, nil
}

type accessPointLifecycleNoop struct{}

func (accessPointLifecycleNoop) Start(_ string) error { return nil }
func (accessPointLifecycleNoop) Stop(_ string) error  { return nil }

func TestC2CConnectWrapsConnectAndRestoreErrors(t *testing.T) {
	resetWifiSeamsForTest(t)
	connectErr := errors.New("connect failure")
	c2cRestoreErr := errors.New("restore failure")

	flow := newC2CModeFlowWithDependencies(c2cModeDependencies{
		c2cModeServiceDependencies: c2cModeServiceDependencies{
			radio: wifiRadioConnectErrorService{
				connectedDevice: "wlan0",
				connectErr:      connectErr,
			},
			accessPoint: accessPointLifecycleNoop{},
			getStoredSSIDs: func(_ []string) {
			},
		},
		c2cModeLifecycleDependencies: c2cModeLifecycleDependencies{
			startResolved: func() error {
				return nil
			},
			stopResolved: func() error {
				return c2cRestoreErr
			},
			rebootSystem: func() error {
				return nil
			},
			pause: func(_ time.Duration) {},
			publishInterval: func(_ string) {
			},
		},
	})

	err := flow.ConnectToNetwork("HomeWiFi", "secret")
	if err == nil {
		t.Fatal("expected C2CConnect to return combined error")
	}
	if !errors.Is(err, connectErr) {
		t.Fatalf("expected connect error in chain, got %v", err)
	}
	if !errors.Is(err, c2cRestoreErr) {
		t.Fatalf("expected C2C mode restore error in chain, got %v", err)
	}
}
