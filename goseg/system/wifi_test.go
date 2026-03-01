package system

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/mdlayher/wifi"
)

func resetWifiSeamsForTest(t *testing.T) {
	t.Helper()
	origExec := execCommandForWiFi
	origRun := runCommandForWiFi
	origIfCheck := ifCheckForWiFi
	origWifiNew := wifiNewClientForWiFi
	t.Cleanup(func() {
		execCommandForWiFi = origExec
		runCommandForWiFi = origRun
		ifCheckForWiFi = origIfCheck
		wifiNewClientForWiFi = origWifiNew
	})
}

func TestGetWifiDeviceParsesAndTrimsOutput(t *testing.T) {
	resetWifiSeamsForTest(t)

	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		if name != "sh" {
			t.Fatalf("unexpected command: %s", name)
		}
		return exec.Command("sh", "-c", "printf 'wlan0\\n wlan1 \\n\\n'")
	}

	devices, err := getWifiDevice()
	if err != nil {
		t.Fatalf("getWifiDevice returned error: %v", err)
	}
	want := []string{"wlan0", "wlan1"}
	if !reflect.DeepEqual(devices, want) {
		t.Fatalf("unexpected wifi devices: got %v want %v", devices, want)
	}
}

func TestGetWifiDeviceReturnsErrorForFailureOrEmptyResult(t *testing.T) {
	resetWifiSeamsForTest(t)

	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	if _, err := getWifiDevice(); err == nil {
		t.Fatal("expected getWifiDevice to fail when command execution fails")
	}

	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf '\\n'")
	}
	if _, err := getWifiDevice(); err == nil {
		t.Fatal("expected getWifiDevice to fail when no devices are listed")
	}
}

func TestPrimaryWifiDeviceReturnsFirstDevice(t *testing.T) {
	resetWifiSeamsForTest(t)

	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'wlan0\\nwlan1\\n'")
	}

	device, err := primaryWifiDevice()
	if err != nil {
		t.Fatalf("primaryWifiDevice returned error: %v", err)
	}
	if device != "wlan0" {
		t.Fatalf("expected primary device wlan0, got %q", device)
	}
}

func TestListWifiSSIDsParsesValidEntriesAndSkipsMalformed(t *testing.T) {
	resetWifiSeamsForTest(t)

	runCommandForWiFi = func(command string, args ...string) (string, error) {
		return strings.Join([]string{
			"a:b:c:d:e:f:g:HomeWiFi",
			"malformed:line",
			"a:b:c:d:e:f:g:",
		}, "\n"), nil
	}

	ssids := ListWifiSSIDs("wlan0")
	want := []string{"HomeWiFi"}
	if !reflect.DeepEqual(ssids, want) {
		t.Fatalf("unexpected ssid list: got %v want %v", ssids, want)
	}
}

func TestIfCheckAndToggleDeviceUseSeams(t *testing.T) {
	resetWifiSeamsForTest(t)

	runCommandForWiFi = func(command string, args ...string) (string, error) {
		if strings.Join(args, " ") == "radio wifi" {
			return "enabled\n", nil
		}
		if strings.Join(args, " ") == "radio wifi off" {
			return "", nil
		}
		if strings.Join(args, " ") == "radio wifi on" {
			return "", nil
		}
		return "", nil
	}
	if !ifCheck() {
		t.Fatal("expected ifCheck to return true when nmcli output contains enabled")
	}

	if err := ToggleDevice("wlan0"); err != nil {
		t.Fatalf("ToggleDevice(off) returned error: %v", err)
	}

	ifCheckForWiFi = func() bool { return false }
	if err := ToggleDevice("wlan0"); err != nil {
		t.Fatalf("ToggleDevice(on) returned error: %v", err)
	}

	runCommandForWiFi = func(string, ...string) (string, error) { return "", errors.New("nmcli error") }
	if ifCheck() {
		t.Fatal("expected ifCheck to return false when command errors")
	}
}

func TestConnectToWifiUsesNmcliAndPropagatesErrors(t *testing.T) {
	resetWifiSeamsForTest(t)

	calls := []string{}
	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return exec.Command("true")
	}
	if err := ConnectToWifi("HomeWiFi", "secret"); err != nil {
		t.Fatalf("ConnectToWifi returned error: %v", err)
	}
	if len(calls) != 1 || !strings.Contains(calls[0], "nmcli dev wifi connect HomeWiFi password secret") {
		t.Fatalf("unexpected nmcli invocation: %v", calls)
	}

	execCommandForWiFi = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	if err := ConnectToWifi("HomeWiFi", "secret"); err == nil {
		t.Fatal("expected ConnectToWifi to return command failure")
	}
}

func TestDisconnectWifiUsesInjectedClientFactoryError(t *testing.T) {
	resetWifiSeamsForTest(t)

	wifiNewClientForWiFi = func() (*wifi.Client, error) {
		return nil, errors.New("client unavailable")
	}
	if err := DisconnectWifi("wlan0"); err == nil {
		t.Fatal("expected DisconnectWifi to return client creation error")
	}
}
