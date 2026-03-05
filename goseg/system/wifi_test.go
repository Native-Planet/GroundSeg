package system

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"groundseg/internal/testruntime"

	"github.com/mdlayher/wifi"
)

func resetWifiSeamsForTest(t *testing.T) {
	t.Helper()
}

func testWifiRuntime(overrides wifiRuntime) wifiRuntime {
	return testruntime.Apply(wifiRuntime{}, func(runtime *wifiRuntime) {
		runtime.ExecCommand = overrides.ExecCommand
		runtime.RunCommand = overrides.RunCommand
		runtime.NewWifiClient = overrides.NewWifiClient
		runtime.ClientInterfacesFn = overrides.ClientInterfacesFn
		runtime.ClientBSSFn = overrides.ClientBSSFn
	})
}

func TestGetWifiDeviceParsesAndTrimsOutput(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			if command != "nmcli" {
				t.Fatalf("unexpected command: %s", command)
			}
			if strings.Join(args, " ") != "-t -f DEVICE,TYPE device status" {
				t.Fatalf("unexpected nmcli args: %q", strings.Join(args, " "))
			}
			return "wlan0:wifo\n wlan1 :wifi\neth0:ethernet\nwlan2:wifi\n", nil
		},
	})

	devices, err := runtime.wifiDevices()
	if err != nil {
		t.Fatalf("wifiDevices returned error: %v", err)
	}
	want := []string{"wlan1", "wlan2"}
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
	if !errors.Is(err, ErrWiFiInterfaceNotFound) {
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
		RunCommand: func(command string, args ...string) (string, error) {
			_ = command
			_ = args
			return "", errors.New("failure")
		},
	})
	if _, err := runtime.wifiDevices(); err == nil {
		t.Fatal("expected wifiDevices to fail when command execution fails")
	}

	runtime = testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			_ = command
			_ = args
			return "\n", nil
		},
	})
	if _, err := runtime.wifiDevices(); err == nil {
		t.Fatal("expected wifiDevices to fail when no devices are listed")
	}
	if _, err := runtime.wifiDevices(); err != nil && !errors.Is(err, ErrWiFiInterfaceNotFound) {
		t.Fatalf("expected wifi interface not found error when no devices are listed, got: %v", err)
	}
}

func TestParseNmcliWifiDeviceRecordRequiresDeviceAndTypeFields(t *testing.T) {
	record, err := parseNmcliWifiDeviceRecord("wlan0", 0)
	if err == nil {
		t.Fatalf("expected malformed wifi device record to fail, got %#v", record)
	}
	if _, ok := err.(wifiNmcliParseError); !ok {
		t.Fatalf("expected parse error, got %T: %v", err, err)
	}

	record, err = parseNmcliWifiDeviceRecord("wlan0:wifi", 1)
	if err != nil {
		t.Fatalf("expected valid wifi device record to parse, got: %v", err)
	}
	if record.name != "wlan0" || record.typ != "wifi" {
		t.Fatalf("unexpected parsed device record: %#v", record)
	}
}

func TestParseNmcliWifiScanResultRecordSkipsMalformedRows(t *testing.T) {
	record, err := parseNmcliWifiScanResultRecord("a:b:c", 0)
	if err == nil {
		t.Fatalf("expected malformed scan row to fail, got %#v", record)
	}
	if _, ok := err.(wifiNmcliParseError); !ok {
		t.Fatalf("expected parse error, got %T: %v", err, err)
	}

	record2, err := parseNmcliWifiScanResultRecord("a:b:c:d:e:f:g:HomeWiFi", 1)
	if err != nil {
		t.Fatalf("expected valid scan record to parse, got: %v", err)
	}
	if record2.ssid != "HomeWiFi" {
		t.Fatalf("unexpected parsed ssid: %#v", record2)
	}
}

func TestParseNmcliWifiDevicesReturnsParseErrorWhenOnlyMalformedRows(t *testing.T) {
	_, err := parseNmcliWifiDevices("malformed-row\na:b")
	if err == nil {
		t.Fatalf("expected parse error for malformed records when no valid devices are found")
	}
	if _, ok := err.(wifiNmcliParseError); !ok {
		t.Fatalf("expected parse error type, got %T: %v", err, err)
	}
}

func TestPrimaryWifiDeviceReturnsFirstDevice(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			_ = command
			_ = args
			return "wlan0:wifi\nwlan1:wifi\n", nil
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
		t.Fatalf("listSSIDs returned error: %v", err)
	}
	want := []string{"HomeWiFi"}
	if !reflect.DeepEqual(ssids, want) {
		t.Fatalf("unexpected ssid list: got %v want %v", ssids, want)
	}
}

func TestListWifiSSIDsSkipsUnderLengthNmcliRows(t *testing.T) {
	resetWifiSeamsForTest(t)

	runtime := testWifiRuntime(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			_ = command
			_ = args
			return strings.Join([]string{
				"malformed",
				"a:b:c",
				"a:b:c:d:e:f:g",
				"",
			}, "\n"), nil
		},
	})

	ssids, err := runtime.listSSIDs("wlan0")
	if err == nil {
		t.Fatal("expected malformed rows to produce a parse error")
	}
	if _, ok := err.(wifiNmcliParseError); !ok {
		t.Fatalf("expected parse error type, got %T: %v", err, err)
	}
	if len(ssids) != 0 {
		t.Fatalf("expected malformed rows to be ignored when parse fails, got %#v", ssids)
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
		t.Fatal("expected listSSIDs to return an error")
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
	wifiEnabled, err := runtime.isWiFiRadioEnabled()
	if err != nil {
		t.Fatalf("ifCheck returned error: %v", err)
	}
	if !wifiEnabled {
		t.Fatal("expected ifCheck to return true when nmcli output contains enabled")
	}

	if err := runtime.toggleDevice("wlan0"); err != nil {
		t.Fatalf("toggleDevice(off) returned error: %v", err)
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
		t.Fatalf("toggleDevice(on) returned error: %v", err)
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
	wifiEnabled, err = runtime.isWiFiRadioEnabled()
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
		t.Fatalf("ConnectToWiFi returned error: %v", err)
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
		t.Fatal("expected ConnectToWiFi to return command failure")
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
		t.Fatal("expected DisconnectWiFi to return client creation error")
	}
}

func TestConstructWifiInfoNilReceiverNoPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("construct wifi info should not panic on nil receiver, got: %v", rec)
		}
	}()
	var service *WiFiRuntimeService
	if err := service.RefreshWiFiInfo(""); err == nil {
		t.Fatal("expected nil receiver error")
	}
}

func TestStartWiFiInfoLoopNilReceiverReturnsError(t *testing.T) {
	var service *WiFiRuntimeService
	if err := service.StartWiFiInfoLoop(context.Background()); err == nil {
		t.Fatal("expected StartWiFiInfoLoop to return nil receiver error")
	}
}

func TestStartWiFiInfoLoopStateNilInitializesAndCanStop(t *testing.T) {
	service := &WiFiRuntimeService{
		runtime: NewWiFiRuntimeWith(wifiRuntime{
			RunCommand: func(command string, args ...string) (string, error) {
				if command == "nmcli" && len(args) == 5 && args[0] == "-t" && args[1] == "-f" && args[2] == "DEVICE,TYPE" && args[3] == "device" && args[4] == "status" {
					return "wlan0:wifi\n", nil
				}
				if command == "nmcli" && len(args) == 2 && args[0] == "radio" && args[1] == "wifi" {
					return "disabled\n", nil
				}
				return "", nil
			},
		}),
	}
	if err := service.StartWiFiInfoLoop(context.Background()); err != nil {
		t.Fatalf("expected StartWiFiInfoLoop to succeed with nil state, got: %v", err)
	}
	if service.state == nil {
		t.Fatal("expected service state to be initialized when nil")
	}
	if err := service.StopWiFiInfoLoop(); err != nil {
		t.Fatalf("expected StopWiFiInfoLoop to return nil, got %v", err)
	}
}

func TestStopWiFiInfoLoopNilReceiverDoesNotPanic(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("expected stop wifi loop to ignore nil receiver, got: %v", rec)
		}
	}()
	var service *WiFiRuntimeService
	if err := service.StopWiFiInfoLoop(); err == nil {
		t.Fatal("expected StopWiFiInfoLoop to return an error for nil receiver")
	}
}

func TestConstructWifiInfoInitializesRuntimeAndState(t *testing.T) {
	service := &WiFiRuntimeService{
		state: &wifiRuntimeState{},
	}
	if err := service.RefreshWiFiInfo(""); err == nil {
		t.Fatal("expected construct wifi info to fail when device is unresolved")
	}
	if service.state == nil {
		t.Fatal("expected service state to be initialized")
	}
	if service.runtime.RunCommand == nil {
		t.Fatal("expected service runtime to be initialized from defaults")
	}
}

func TestConstructWifiInfoRefreshesWithCustomRuntime(t *testing.T) {
	state := &wifiRuntimeState{}
	setWiFiDevice("wlan0", state)
	runtime := NewWiFiRuntimeWith(wifiRuntime{
		RunCommand: func(command string, args ...string) (string, error) {
			switch {
			case command == "nmcli" && strings.Join(args, " ") == "radio wifi":
				return "enabled", nil
			case command == "nmcli" && strings.Contains(strings.Join(args, " "), "dev wifi list ifname wlan0"):
				return "a:b:c:d:e:f:g:HomeWiFi", nil
			}
			return "", errors.New("unexpected command")
		},
		NewWifiClient: func() (*wifi.Client, error) {
			return &wifi.Client{}, nil
		},
		ClientInterfacesFn: func(*wifi.Client) ([]*wifi.Interface, error) {
			return []*wifi.Interface{{Name: "wlan0", Type: wifi.InterfaceTypeStation}}, nil
		},
		ClientBSSFn: func(*wifi.Client, *wifi.Interface) (*wifi.BSS, error) {
			return &wifi.BSS{SSID: "HomeWiFi"}, nil
		},
	})
	service := &WiFiRuntimeService{
		runtime: runtime,
		state:   state,
	}
	if err := service.RefreshWiFiInfo(""); err != nil {
		t.Fatalf("expected construct info to succeed: %v", err)
	}
	info := WifiInfoSnapshot(state)
	if !info.Status {
		t.Fatalf("expected refreshed status to be true, got %v", info.Status)
	}
	if info.Active != "HomeWiFi" {
		t.Fatalf("expected active SSID to be HomeWiFi, got %q", info.Active)
	}
	if len(info.Networks) == 0 || info.Networks[0] != "HomeWiFi" {
		t.Fatalf("expected networks to include HomeWiFi, got %v", info.Networks)
	}
}

func TestNewWiFiRuntimeServiceMergesPartialOverridesWithDefaults(t *testing.T) {
	resetWifiSeamsForTest(t)
	runtime := NewWiFiRuntimeWith(wifiRuntime{
		ExecCommand: func(name string, args ...string) *exec.Cmd {
			return exec.Command(name, args...)
		},
	})
	service := &WiFiRuntimeService{
		runtime: runtime,
	}
	resolved, _, err := service.prepareForUse()
	if err != nil {
		t.Fatalf("prepareForUse returned error: %v", err)
	}
	if resolved.ExecCommand == nil || resolved.RunCommand == nil || resolved.RunNmcliFn == nil || resolved.NewWifiClient == nil ||
		resolved.ClientInterfacesFn == nil || resolved.ClientBSSFn == nil || resolved.WifiInfoTicker == nil {
		t.Fatal("expected partial wifi runtime override to be completed with defaults")
	}
}
