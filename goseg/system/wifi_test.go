package system

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mdlayher/wifi"
	"groundseg/protocol/actions"
)

func resetWifiSeamsForTest(t *testing.T) {
	t.Helper()
}

func testWifiRuntime(overrides wifiRuntime) wifiRuntime {
	runtime := wifiRuntime{}
	runtime.ExecCommand = overrides.ExecCommand
	runtime.RunCommand = overrides.RunCommand
	runtime.NewWifiClient = overrides.NewWifiClient
	runtime.ClientInterfacesFn = overrides.ClientInterfacesFn
	runtime.ClientBSSFn = overrides.ClientBSSFn
	return runtime
}

func TestGetWifiDeviceParsesAndTrimsOutput(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			if name != "sh" {
				t.Fatalf("unexpected command: %s", name)
			}
			return exec.Command("sh", "-c", "printf 'wlan0\\n wlan1 \\n\\n'")
		},
	})

	devices, err := runtime.wifiDevices()
	if err != nil {
		t.Fatalf("getWifiDevice returned error: %v", err)
	}
	want := []string{"wlan0", "wlan1"}
	if !reflect.DeepEqual(devices, want) {
		t.Fatalf("unexpected wifi devices: got %v want %v", devices, want)
	}
}

func TestGetConnectedSSIDReturnsErrorForMissingInterface(t *testing.T) {
	resetWifiSeamsForTest(t)

	called := 0
	bssCalled := 0
	runtime := testWifiRuntime(wifiRuntime{
		ClientInterfacesFn: func(c *wifi.Client) ([]*wifi.Interface, error) {
			_ = c
			called++
			return []*wifi.Interface{{Name: "wlan1", Type: wifi.InterfaceTypeStation}}, nil
		},
		ClientBSSFn: func(c *wifi.Client, iface *wifi.Interface) (*wifi.BSS, error) {
			_ = c
			_ = iface
			bssCalled++
			return &wifi.BSS{}, nil
		},
	})

	ssid, err := runtime.connectedSSID(&wifi.Client{}, "wlan0")
	if err == nil {
		t.Fatal("expected getConnectedSSID to fail when interface is missing")
	}
	if !errors.Is(err, ErrWifiInterfaceNotFound) {
		t.Fatalf("expected wifi interface not found error, got: %v", err)
	}
	if ssid != "" {
		t.Fatalf("expected empty ssid on missing interface, got %q", ssid)
	}
	if called != 1 {
		t.Fatalf("unexpected direct interface call count: %d", called)
	}
	if bssCalled != 0 {
		t.Fatalf("expected BSS lookup to be skipped when interface is missing: %d", bssCalled)
	}
}

func TestGetWifiDeviceReturnsErrorForFailureOrEmptyResult(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			_ = args
			return exec.Command("false")
		},
	})
	if _, err := runtime.wifiDevices(); err == nil {
		t.Fatal("expected getWifiDevice to fail when command execution fails")
	}

	runtime = testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			_ = name
			_ = args
			return exec.Command("sh", "-c", "printf '\\n'")
		},
	})
	if _, err := runtime.wifiDevices(); err == nil {
		t.Fatal("expected getWifiDevice to fail when no devices are listed")
	}
	if _, err := runtime.wifiDevices(); err != nil && !errors.Is(err, ErrWifiInterfaceNotFound) {
		t.Fatalf("expected wifi interface not found error when no devices are listed, got: %v", err)
	}
}

func TestPrimaryWifiDeviceReturnsFirstDevice(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			_ = name
			_ = args
			return exec.Command("sh", "-c", "printf 'wlan0\\nwlan1\\n'")
		},
	})

	device, err := runtime.primaryWifiDevice()
	if err != nil {
		t.Fatalf("primaryWifiDevice returned error: %v", err)
	}
	if device != "wlan0" {
		t.Fatalf("expected primary device wlan0, got %q", device)
	}
}

func TestListWifiSSIDsParsesValidEntriesAndSkipsMalformed(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			_ = command
			_ = args
			return strings.Join([]string{
				"a:b:c:d:e:f:g:HomeWiFi",
				"malformed:line",
				"a:b:c:d:e:f:g:",
			}, "\n"), nil
		},
	})

	ssids, err := runtime.listSSIDs("wlan0")
	if err != nil {
		t.Fatalf("ListWifiSSIDs returned error: %v", err)
	}
	want := []string{"HomeWiFi"}
	if !reflect.DeepEqual(ssids, want) {
		t.Fatalf("unexpected ssid list: got %v want %v", ssids, want)
	}
}

func TestListWifiSSIDsPropagatesScanFailures(t *testing.T) {
	resetWifiSeamsForTest(t)
	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(string, ...string) (string, error) {
			return "", errors.New("nmcli down")
		},
	})
	if _, err := runtime.listSSIDs("wlan0"); err == nil {
		t.Fatal("expected ListWifiSSIDs to return an error")
	}
}

func TestIfCheckAndToggleDeviceUseSeams(t *testing.T) {
	resetWifiSeamsForTest(t)

	calls := []string{}
	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			call := command + " " + strings.Join(args, " ")
			calls = append(calls, strings.TrimSpace(call))
			switch {
			case command == "nmcli" && strings.Join(args, " ") == "radio wifi":
				return "enabled\n", nil
			case command == "ip" && strings.Join(args, " ") == "link set wlan0 down":
				return "", nil
			case command == "ip" && strings.Join(args, " ") == "link set wlan0 up":
				return "", nil
			}
			return "", errors.New("unexpected command: " + call)
		},
	})
	wifiEnabled, err := runtime.ifCheck()
	if err != nil {
		t.Fatalf("ifCheck returned error: %v", err)
	}
	if !wifiEnabled {
		t.Fatal("expected ifCheck to return true when nmcli output contains enabled")
	}

	if err := runtime.toggleDevice("wlan0"); err != nil {
		t.Fatalf("ToggleDevice(off) returned error: %v", err)
	}

	runtime = testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			call := command + " " + strings.Join(args, " ")
			calls = append(calls, strings.TrimSpace(call))
			switch {
			case command == "nmcli" && strings.Join(args, " ") == "radio wifi":
				return "disabled\n", nil
			case command == "ip" && strings.Join(args, " ") == "link set wlan0 up":
				return "", nil
			}
			return "", errors.New("unexpected command: " + call)
		},
	})
	if err := runtime.toggleDevice("wlan0"); err != nil {
		t.Fatalf("ToggleDevice(on) returned error: %v", err)
	}

	wantCalls := map[string]struct{}{
		"nmcli radio wifi":       {},
		"ip link set wlan0 down": {},
		"ip link set wlan0 up":   {},
	}
	for _, call := range calls {
		if _, ok := wantCalls[call]; !ok {
			t.Fatalf("unexpected command in wifi seam test: %q", call)
		}
	}

	runtime = testWifiRuntime(wifiRuntime{
		RunCommand: func(string, ...string) (string, error) { return "", errors.New("nmcli error") },
	})
	wifiEnabled, err = runtime.ifCheck()
	if err == nil {
		t.Fatal("expected ifCheck to return error when command fails")
	}
	if wifiEnabled {
		t.Fatal("expected ifCheck to default to false when command fails")
	}
}

func TestConnectToWifiUsesNmcliAndPropagatesErrors(t *testing.T) {
	resetWifiSeamsForTest(t)

	calls := []string{}
	runtime := testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			calls = append(calls, name+" "+strings.Join(args, " "))
			return exec.Command("true")
		},
		RunCommand: func(command string, args ...string) (string, error) {
			call := command + " " + strings.Join(args, " ")
			calls = append(calls, strings.TrimSpace(call))
			return "", errors.New("unexpected command: " + call)
		},
	})
	if err := runtime.connect("HomeWiFi", "secret"); err != nil {
		t.Fatalf("ConnectToWifi returned error: %v", err)
	}
	if len(calls) != 1 || !strings.Contains(calls[0], "nmcli dev wifi connect HomeWiFi password secret") {
		t.Fatalf("unexpected nmcli invocation: %v", calls)
	}

	runtime = testWifiRuntime(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			calls = []string{}
			calls = append(calls, name+" "+strings.Join(args, " "))
			return exec.Command("false")
		},
		RunCommand: func(command string, args ...string) (string, error) {
			call := command + " " + strings.Join(args, " ")
			calls = append(calls, strings.TrimSpace(call))
			return "", errors.New("unexpected command: " + call)
		},
	})
	if err := runtime.connect("HomeWiFi", "secret"); err == nil {
		t.Fatal("expected ConnectToWifi to return command failure")
	}
}

func TestDisconnectWifiUsesInjectedClientFactoryError(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		NewWifiClient: func() (*wifi.Client, error) {
			return nil, errors.New("client unavailable")
		},
	})
	if err := runtime.disconnect("wlan0"); err == nil {
		t.Fatal("expected DisconnectWifi to return client creation error")
	}
}

func TestParseC2CAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	action, err := actions.ParseC2CAction(string(c2cActionConnect))
	if err != nil {
		t.Fatalf("expected connect action to parse: %v", err)
	}
	if action != c2cActionConnect {
		t.Fatalf("unexpected action: %v", action)
	}
}

func TestParseC2CActionRejectsUnsupportedAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	if _, err := actions.ParseC2CAction("unsupported"); err == nil {
		t.Fatal("expected parse failure for unsupported action")
	}
}

func TestProcessMessageRoutesSupportedAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	adapter := newCaptiveTransportAdapter(defaultC2CServiceDeps())
	var callCount int
	adapter.processC2CMessage = func(msg []byte) error {
		callCount++
		if string(msg) != `{"type":"c2c","payload":{"action":"connect","ssid":"HomeWiFi","password":"secret"}}` {
			t.Fatalf("unexpected payload: %s", msg)
		}
		return nil
	}

	msg := `{"type":"c2c","payload":{"action":"connect","ssid":"HomeWiFi","password":"secret"}}`
	if err := adapter.processMessage([]byte(msg)); err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected c2c message processor to run once; got %d", callCount)
	}
}

func TestProcessMessageRejectsUnsupportedAction(t *testing.T) {
	resetWifiSeamsForTest(t)
	adapter := newCaptiveTransportAdapter(defaultC2CServiceDeps())

	msg := `{"type":"c2c","payload":{"action":"unsupported","ssid":"HomeWiFi","password":"secret"}}`
	processErr := adapter.processMessage([]byte(msg))
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

	msg := `{"type":"auth","payload":{"action":"connect","ssid":"HomeWiFi","password":"secret"}}`
	processErr := adapter.processMessage([]byte(msg))
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
	msg := `{"type":"c2c","payload":{"action":"connect","ssid":"HomeWiFi","password":"secret"}}`
	if err := processC2CMessageForAdapterWithDeps([]byte(msg), deps); err != nil {
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

	msg := `{"type":"c2c","payload":{"action":"connect","ssid":"HomeWiFi","password":"secret"}}`
	deps := c2cServiceDeps{
		connectToWiFi: nil,
	}

	err := processC2CMessageForAdapterWithDeps([]byte(msg), deps)
	if err == nil {
		t.Fatal("expected processC2CMessageForAdapter to fail when dependencies are nil")
	}
	if !strings.Contains(err.Error(), "c2c service is required") {
		t.Fatalf("expected service requirement error, got %v", err)
	}
}

func TestC2CActionBindingsCoverKnownActions(t *testing.T) {
	resetWifiSeamsForTest(t)
	foundConnect := false
	for _, action := range supportedC2CActions() {
		if _, err := actions.ParseC2CAction(string(action)); err != nil {
			t.Fatalf("action %v should be supported by parser", action)
		}
		if action == c2cActionConnect {
			foundConnect = true
		}
	}
	if !foundConnect {
		t.Fatalf("expected connect action binding")
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
