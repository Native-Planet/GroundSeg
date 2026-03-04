package system

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/handler/systemsvc"
	"groundseg/structs"
	"groundseg/system"
)

func resetSystemSeams() {
	confForSystemHandler = config.Conf
	stopContainerForSystemHandler = orchestration.StopContainerByName
	updateConfTypedForSystemHandler = config.UpdateConfTyped
	withPenpaiAllowForSystemHandler = config.WithPenpaiAllow
	loadLlamaForSystemHandler = orchestration.LoadLlama
	withGracefulExitForSystemHandler = config.WithGracefulExit
	execCommandForSystemHandler = func(name string, args ...string) systemsvc.CommandRunner {
		return exec.Command(name, args...)
	}
	configureSwapForSystemHandler = system.ConfigureSwap
	withSwapValForSystemHandler = config.WithSwapVal
	runUpgradeForSystemHandler = system.RunUpgrade
	toggleDeviceForSystemHandler = func(dev string) error {
		return system.NewWiFiRuntimeService().ToggleDevice(dev)
	}
	connectToWifiForSystemHandler = func(ssid, password string) error {
		return system.NewWiFiRuntimeService().ConnectToWifi(ssid, password)
	}
	publishSystemTransitionForSystemHandler = func(_ context.Context, transition structs.SystemTransition) error {
		_ = events.DefaultEventRuntime().PublishSystemTransition(context.Background(), transition)
		return nil
	}
	sleepForSystemHandler = time.Sleep
}

func systemMessage(t *testing.T, action string, opts ...func(*structs.WsSystemAction)) []byte {
	t.Helper()
	payload := structs.WsSystemPayload{
		Payload: structs.WsSystemAction{Action: action},
	}
	for _, opt := range opts {
		opt(&payload.Payload)
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return msg
}

func TestSystemHandlerValidationAndUnknown(t *testing.T) {
	t.Cleanup(resetSystemSeams)
	if err := SystemHandler([]byte("{invalid")); err == nil {
		t.Fatalf("expected unmarshal error")
	}
	if err := SystemHandler(systemMessage(t, "unknown")); err == nil {
		t.Fatalf("expected unknown action error")
	}
}

func TestSystemHandlerTogglePenpaiFeature(t *testing.T) {
	t.Cleanup(resetSystemSeams)

	stopped := []string{}
	stopContainerForSystemHandler = func(name string) error {
		stopped = append(stopped, name)
		return nil
	}
	updateCalls := 0
	updateConfTypedForSystemHandler = func(...config.ConfUpdateOption) error {
		updateCalls++
		return nil
	}
	confForSystemHandler = func() structs.SysConfig {
		return structs.SysConfig{Penpai: structs.PenpaiConfig{PenpaiAllow: true}}
	}
	if err := SystemHandler(systemMessage(t, "toggle-penpai-feature")); err != nil {
		t.Fatalf("toggle disable failed: %v", err)
	}
	if !reflect.DeepEqual(stopped, []string{"llama-gpt-api", "llama-gpt-ui"}) || updateCalls != 1 {
		t.Fatalf("unexpected disable flow: stopped=%v updates=%d", stopped, updateCalls)
	}

	loadCalled := false
	loadLlamaForSystemHandler = func() error { loadCalled = true; return nil }
	confForSystemHandler = func() structs.SysConfig {
		return structs.SysConfig{Penpai: structs.PenpaiConfig{PenpaiAllow: false}}
	}
	if err := SystemHandler(systemMessage(t, "toggle-penpai-feature")); err != nil {
		t.Fatalf("toggle enable failed: %v", err)
	}
	if !loadCalled || updateCalls != 2 {
		t.Fatalf("expected llama load and config update on enable, load=%v updates=%d", loadCalled, updateCalls)
	}
}

func TestSystemHandlerGroundsegPowerAndSwap(t *testing.T) {
	t.Cleanup(resetSystemSeams)
	origDebug := config.DebugMode()
	config.SetDebugMode(true)
	t.Cleanup(func() { config.SetDebugMode(origDebug) })

	updateCalls := 0
	updateConfTypedForSystemHandler = func(...config.ConfUpdateOption) error {
		updateCalls++
		return nil
	}
	execCalled := 0
	execCommandForSystemHandler = func(string, ...string) systemsvc.CommandRunner {
		execCalled++
		return exec.Command("true")
	}

	if err := SystemHandler(systemMessage(t, "groundseg", func(a *structs.WsSystemAction) { a.Command = "restart" })); err != nil {
		t.Fatalf("groundseg restart failed: %v", err)
	}
	if err := SystemHandler(systemMessage(t, "groundseg", func(a *structs.WsSystemAction) { a.Command = "bad" })); err == nil {
		t.Fatalf("expected invalid groundseg command error")
	}
	if err := SystemHandler(systemMessage(t, "power", func(a *structs.WsSystemAction) { a.Command = "shutdown" })); err != nil {
		t.Fatalf("power shutdown failed: %v", err)
	}
	if err := SystemHandler(systemMessage(t, "power", func(a *structs.WsSystemAction) { a.Command = "bad" })); err == nil {
		t.Fatalf("expected invalid power command error")
	}
	if execCalled != 0 {
		t.Fatalf("expected no exec invocations in debug mode, got %d", execCalled)
	}

	confForSystemHandler = func() structs.SysConfig {
		return structs.SysConfig{Runtime: structs.RuntimeConfig{SwapFile: "/tmp/swapfile"}}
	}
	configureCalled := false
	configureSwapForSystemHandler = func(file string, value int) error {
		configureCalled = (file == "/tmp/swapfile" && value == 4)
		return nil
	}
	sleepForSystemHandler = func(time.Duration) {}
	if err := SystemHandler(systemMessage(t, "modify-swap", func(a *structs.WsSystemAction) { a.Value = 4 })); err != nil {
		t.Fatalf("modify-swap failed: %v", err)
	}
	if !configureCalled {
		t.Fatalf("expected ConfigureSwap invocation")
	}
}

func TestSystemHandlerUpdateWifiAndWifiConnectTransitions(t *testing.T) {
	t.Cleanup(resetSystemSeams)
	runUpgradeCalled := false
	runUpgradeForSystemHandler = func() error { runUpgradeCalled = true; return nil }
	if err := SystemHandler(systemMessage(t, "update", func(a *structs.WsSystemAction) { a.Update = "linux" })); err != nil {
		t.Fatalf("update action failed: %v", err)
	}
	if !runUpgradeCalled {
		t.Fatalf("expected linux upgrade call")
	}

	toggleCalled := false
	toggleDeviceForSystemHandler = func(string) error { toggleCalled = true; return nil }
	if err := SystemHandler(systemMessage(t, "wifi-toggle")); err != nil {
		t.Fatalf("wifi-toggle failed: %v", err)
	}
	if !toggleCalled {
		t.Fatalf("expected wifi toggle call")
	}

	sleepForSystemHandler = func(time.Duration) {}
	var events []string
	publishSystemTransitionForSystemHandler = func(_ context.Context, trans structs.SystemTransition) error {
		if trans.Type == "wifiConnect" {
			events = append(events, trans.Event)
		}
		return nil
	}
	connectToWifiForSystemHandler = func(string, string) error { return errors.New("auth failed") }
	err := SystemHandler(systemMessage(t, "wifi-connect", func(a *structs.WsSystemAction) {
		a.SSID = "wifi"
		a.Password = "pw"
	}))
	if err == nil {
		t.Fatalf("expected wifi-connect error")
	}
	if !reflect.DeepEqual(events, []string{"connecting", "error", ""}) {
		t.Fatalf("unexpected wifi-connect error transitions: %v", events)
	}

	events = nil
	connectToWifiForSystemHandler = func(string, string) error { return nil }
	if err := SystemHandler(systemMessage(t, "wifi-connect", func(a *structs.WsSystemAction) {
		a.SSID = "wifi"
		a.Password = "pw"
	})); err != nil {
		t.Fatalf("wifi-connect success failed: %v", err)
	}
	if !reflect.DeepEqual(events, []string{"connecting", "success", ""}) {
		t.Fatalf("unexpected wifi-connect success transitions: %v", events)
	}
}
