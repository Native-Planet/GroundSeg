package systemsvc

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
	"groundseg/system"
)

type fakeCommand struct {
	runErr error
}

func (c *fakeCommand) Run() error {
	return c.runErr
}

func TestHandleSystemMalformedPayload(t *testing.T) {
	if err := HandleSystem([]byte("{"), SystemDependencies{}); err == nil {
		t.Fatal("expected malformed JSON to fail")
	}
}

func TestHandleSystemTogglePenpaiFeature(t *testing.T) {
	var stopped []string
	var patch *config.ConfPatch

	deps := SystemDependencies{
		Conf: func() structs.SysConfig {
			return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiAllow: true}}
		},
		StopContainerByName: func(name string) error {
			stopped = append(stopped, name)
			return nil
		},
		UpdateConfTyped: func(opts ...config.ConfUpdateOption) error {
			p := &config.ConfPatch{}
			for _, opt := range opts {
				opt(p)
			}
			patch = p
			return nil
		},
		WithPenpaiAllow: config.WithPenpaiAllow,
		LoadLlama:       func() error { t.Fatal("LoadLlama should not run when disabling penpai feature"); return nil },
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"toggle-penpai-feature"}}`), deps); err != nil {
		t.Fatalf("expected disabling feature to succeed: %v", err)
	}
	if len(stopped) != 2 {
		t.Fatalf("expected both containers stopped, got %d", len(stopped))
	}
	if patch == nil || patch.PenpaiAllow == nil || *patch.PenpaiAllow {
		t.Fatalf("expected penpai allow patch to set false, got %#v", patch)
	}

	deps = SystemDependencies{
		Conf: func() structs.SysConfig {
			return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiAllow: false}}
		},
		UpdateConfTyped: func(opts ...config.ConfUpdateOption) error {
			p := &config.ConfPatch{}
			for _, opt := range opts {
				opt(p)
			}
			patch = p
			return nil
		},
		WithPenpaiAllow: config.WithPenpaiAllow,
		LoadLlama:       func() error { return nil },
	}
	if err := HandleSystem([]byte(`{"payload":{"action":"toggle-penpai-feature"}}`), deps); err != nil {
		t.Fatalf("expected enabling feature to succeed: %v", err)
	}
	if patch == nil || patch.PenpaiAllow == nil || !*patch.PenpaiAllow {
		t.Fatalf("expected penpai allow patch to set true, got %#v", patch)
	}
}

func TestHandleSystemGroundSegAndPower(t *testing.T) {
	runCount := 0
	updatedGraceful := 0
	deps := SystemDependencies{
		Conf:             func() structs.SysConfig { return structs.SysConfig{} },
		UpdateConfTyped:  func(opts ...config.ConfUpdateOption) error { updatedGraceful++; return nil },
		WithGracefulExit: config.WithGracefulExit,
		ExecCommand: func(_ string, _ ...string) CommandRunner {
			runCount++
			return &fakeCommand{runErr: errors.New("command failed")}
		},
		IsDebugMode:             true,
		WithPenpaiAllow:         config.WithPenpaiAllow,
		WithSwapVal:             config.WithSwapVal,
		RunUpgrade:              func() error { return nil },
		ConfigureSwap:           func(_ string, _ int) error { return nil },
		ToggleDevice:            func(_ string) error { return nil },
		ConnectToWifi:           func(_, _ string) error { return nil },
		PublishSystemTransition: func(structs.SystemTransition) {},
		Sleep:                   func(time.Duration) {},
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"groundseg","command":"restart"}}`), deps); err != nil {
		t.Fatalf("expected debug-mode restart to succeed: %v", err)
	}
	if runCount != 0 {
		t.Fatalf("expected no command in debug mode, got %d", runCount)
	}

	deps.IsDebugMode = false
	runCount = 0
	if err := HandleSystem([]byte(`{"payload":{"action":"groundseg","command":"restart"}}`), deps); err == nil {
		t.Fatal("expected groundseg restart to fail when command fails")
	}
	if runCount != 1 {
		t.Fatalf("expected one groundseg command, got %d", runCount)
	}

	runCount = 0
	if err := HandleSystem([]byte(`{"payload":{"action":"power","command":"shutdown"}}`), deps); err == nil {
		t.Fatal("expected power shutdown to fail when command fails")
	}
	if runCount != 1 {
		t.Fatalf("expected one shutdown command, got %d", runCount)
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"power","command":"oops"}}`), deps); err == nil {
		t.Fatal("expected unrecognized power command to fail")
	}
	if updatedGraceful != 3 {
		t.Fatalf("expected graceful exit set for debug groundseg, groundseg, and power actions, got %d", updatedGraceful)
	}
}

func TestHandleSystemSwapWifiAndUpdate(t *testing.T) {
	configureCalls := []int{}
	updatedSwaps := []int{}
	transitionCount := 0
	sleepCount := 0
	toggleArg := ""
	upgradeCount := 0
	deps := SystemDependencies{
		Conf: func() structs.SysConfig {
			return structs.SysConfig{RuntimeConfig: structs.RuntimeConfig{SwapFile: "/tmp/swapfile"}}
		},
		ConfigureSwap: func(_ string, value int) error { configureCalls = append(configureCalls, value); return nil },
		UpdateConfTyped: func(opts ...config.ConfUpdateOption) error {
			p := &config.ConfPatch{}
			for _, opt := range opts {
				opt(p)
			}
			if p.SwapVal != nil {
				updatedSwaps = append(updatedSwaps, *p.SwapVal)
			}
			return nil
		},
		WithSwapVal:             config.WithSwapVal,
		WithPenpaiAllow:         config.WithPenpaiAllow,
		WithGracefulExit:        config.WithGracefulExit,
		ExecCommand:             func(_ string, _ ...string) CommandRunner { return &fakeCommand{} },
		IsDebugMode:             false,
		RunUpgrade:              func() error { upgradeCount++; return nil },
		ToggleDevice:            func(dev string) error { toggleArg = dev; return nil },
		ConnectToWifi:           func(_, _ string) error { return nil },
		PublishSystemTransition: func(_ structs.SystemTransition) { transitionCount++ },
		Sleep:                   func(time.Duration) { sleepCount++ },
	}

	deps.ConfigureSwap = func(_ string, _ int) error { return errors.New("set swap failed") }
	if err := HandleSystem([]byte(`{"payload":{"action":"modify-swap","value":4}}`), deps); err == nil {
		t.Fatal("expected configure-swap failure to surface")
	}
	if len(configureCalls) != 0 {
		t.Fatalf("expected no successful swap configuration call for failed request, got %v", configureCalls)
	}

	deps.ConfigureSwap = func(_ string, value int) error { configureCalls = append(configureCalls, value); return nil }
	if err := HandleSystem([]byte(`{"payload":{"action":"modify-swap","value":8}}`), deps); err != nil {
		t.Fatalf("expected modify-swap success: %v", err)
	}
	if len(updatedSwaps) != 1 || updatedSwaps[0] != 8 {
		t.Fatalf("expected swap val update to 8, got %v", updatedSwaps)
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"update","update":"linux"}}`), deps); err != nil {
		t.Fatalf("expected linux update to succeed: %v", err)
	}
	if upgradeCount != 1 {
		t.Fatalf("expected one upgrade run, got %d", upgradeCount)
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"wifi-toggle"}}`), deps); err != nil {
		t.Fatalf("expected wifi-toggle to succeed: %v", err)
	}
	if toggleArg != system.WiFiDevice() {
		t.Fatalf("expected toggle to use system.WiFiDevice (%s), got %s", system.WiFiDevice(), toggleArg)
	}

	if err := HandleSystem([]byte(`{"payload":{"action":"wifi-connect","ssid":"net","password":"pw"}}`), deps); err != nil {
		t.Fatalf("expected wifi-connect success: %v", err)
	}
	if transitionCount != 3 {
		t.Fatalf("expected 3 transitions for successful wifi-connect, got %d", transitionCount)
	}
	if sleepCount != 1 {
		t.Fatalf("expected one sleep call on successful wifi-connect, got %d", sleepCount)
	}

	deps.ConnectToWifi = func(_ string, _ string) error { return errors.New("wifi failure") }
	transitionCount = 0
	sleepCount = 0
	if err := HandleSystem([]byte(`{"payload":{"action":"wifi-connect","ssid":"net","password":"pw"}}`), deps); err == nil {
		t.Fatal("expected wifi-connect failure")
	}
	if transitionCount != 3 {
		t.Fatalf("expected 3 transitions on failed wifi-connect, got %d", transitionCount)
	}
	if sleepCount != 1 {
		t.Fatalf("expected one sleep call on failed wifi-connect, got %d", sleepCount)
	}
}

func TestHandleSystemUnrecognizedAction(t *testing.T) {
	deps := SystemDependencies{
		Conf:                    func() structs.SysConfig { return structs.SysConfig{} },
		UpdateConfTyped:         func(...config.ConfUpdateOption) error { return nil },
		WithPenpaiAllow:         config.WithPenpaiAllow,
		WithSwapVal:             config.WithSwapVal,
		WithGracefulExit:        config.WithGracefulExit,
		RunUpgrade:              func() error { return nil },
		ConfigureSwap:           func(string, int) error { return nil },
		ToggleDevice:            func(string) error { return nil },
		ConnectToWifi:           func(string, string) error { return nil },
		PublishSystemTransition: func(structs.SystemTransition) {},
		Sleep:                   func(time.Duration) {},
	}
	if err := HandleSystem([]byte(`{"payload":{"action":"unknown"}}`), deps); err == nil {
		t.Fatal("expected unrecognized system action to fail")
	}
}

func TestHandlePenpaiMalformedPayload(t *testing.T) {
	if err := HandlePenpai([]byte("{"), PenpaiDependencies{}); err == nil {
		t.Fatal("expected malformed JSON to fail")
	}
}

func TestHandlePenpaiToggle(t *testing.T) {
	stopCalls := 0
	startCalls := 0
	updatedRunning := []bool{}

	deps := PenpaiDependencies{
		Unmarshal: json.Unmarshal,
		Conf: func() structs.SysConfig {
			return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiRunning: true}}
		},
		StopContainerByName: func(name string) error {
			stopCalls++
			return nil
		},
		StartContainerByName: func(_, _ string) (structs.ContainerState, error) {
			startCalls++
			return structs.ContainerState{}, nil
		},
		UpdateContainerState: func(string, structs.ContainerState) {},
		UpdateConfTyped: func(opts ...config.ConfUpdateOption) error {
			p := &config.ConfPatch{}
			for _, opt := range opts {
				opt(p)
			}
			if p.PenpaiRunning != nil {
				updatedRunning = append(updatedRunning, *p.PenpaiRunning)
			}
			return nil
		},
		WithPenpaiRunning: config.WithPenpaiRunning,
		WithPenpaiActive:  config.WithPenpaiActive,
		WithPenpaiCores:   config.WithPenpaiCores,
		DeleteContainer:   func(string) error { return nil },
		NumCPU:            func() int { return 8 },
	}
	if err := HandlePenpai([]byte(`{"payload":{"action":"toggle"}}`), deps); err != nil {
		t.Fatalf("expected first toggle to stop containers: %v", err)
	}
	if stopCalls != 2 {
		t.Fatalf("expected two stop calls, got %d", stopCalls)
	}
	if len(updatedRunning) != 1 || updatedRunning[0] {
		t.Fatalf("expected running=false after first toggle, got %v", updatedRunning)
	}

	deps.Conf = func() structs.SysConfig {
		return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiRunning: false}}
	}
	if err := HandlePenpai([]byte(`{"payload":{"action":"toggle"}}`), deps); err != nil {
		t.Fatalf("expected second toggle to start container: %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("expected one start call, got %d", startCalls)
	}
	if len(updatedRunning) != 2 || !updatedRunning[1] {
		t.Fatalf("expected running=true after second toggle, got %v", updatedRunning)
	}
}

func TestHandlePenpaiSetModel(t *testing.T) {
	deleteCalls := 0
	restarts := 0
	deps := PenpaiDependencies{
		Unmarshal: json.Unmarshal,
		Conf: func() structs.SysConfig {
			return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiRunning: true}}
		},
		StopContainerByName: func(string) error { return nil },
		StartContainerByName: func(_ string, _ string) (structs.ContainerState, error) {
			restarts++
			return structs.ContainerState{}, nil
		},
		UpdateContainerState: func(string, structs.ContainerState) {},
		UpdateConfTyped: func(opts ...config.ConfUpdateOption) error {
			p := &config.ConfPatch{}
			for _, opt := range opts {
				opt(p)
			}
			return nil
		},
		WithPenpaiRunning: config.WithPenpaiRunning,
		WithPenpaiActive:  config.WithPenpaiActive,
		WithPenpaiCores:   config.WithPenpaiCores,
		DeleteContainer:   func(_ string) error { deleteCalls++; return nil },
		NumCPU:            func() int { return 8 },
	}

	if err := HandlePenpai([]byte(`{"payload":{"action":"set-model","model":"mistral"}}`), deps); err != nil {
		t.Fatalf("expected set-model to succeed: %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("expected one delete call, got %d", deleteCalls)
	}
	if restarts != 1 {
		t.Fatalf("expected restart when model set and running, got %d", restarts)
	}

	deps.Conf = func() structs.SysConfig {
		return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiRunning: false}}
	}
	restarts = 0
	if err := HandlePenpai([]byte(`{"payload":{"action":"set-model","model":"mistral"}}`), deps); err != nil {
		t.Fatalf("expected set-model to succeed when not running: %v", err)
	}
	if restarts != 0 {
		t.Fatalf("did not expect restart when not running, got %d", restarts)
	}
}

func TestHandlePenpaiSetCoresValidation(t *testing.T) {
	deps := PenpaiDependencies{
		Unmarshal: json.Unmarshal,
		Conf: func() structs.SysConfig {
			return structs.SysConfig{PenpaiConfig: structs.PenpaiConfig{PenpaiRunning: true}}
		},
		StopContainerByName:  func(string) error { return nil },
		StartContainerByName: func(_ string, _ string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
		UpdateContainerState: func(string, structs.ContainerState) {},
		UpdateConfTyped:      func(opts ...config.ConfUpdateOption) error { return nil },
		WithPenpaiRunning:    config.WithPenpaiRunning,
		WithPenpaiActive:     config.WithPenpaiActive,
		WithPenpaiCores:      config.WithPenpaiCores,
		DeleteContainer:      func(_ string) error { return nil },
		NumCPU:               func() int { return 4 },
	}

	if err := HandlePenpai([]byte(`{"payload":{"action":"set-cores","cores":0}}`), deps); err == nil {
		t.Fatal("expected zero cores to fail")
	}
	if err := HandlePenpai([]byte(`{"payload":{"action":"set-cores","cores":4}}`), deps); err == nil {
		t.Fatal("expected >=cpu cores to fail")
	}
	if err := HandlePenpai([]byte(`{"payload":{"action":"set-cores","cores":2}}`), deps); err != nil {
		t.Fatalf("expected valid core count to succeed: %v", err)
	}
}

func TestHandlePenpaiUnrecognizedActionAndRemoveNoOp(t *testing.T) {
	deps := PenpaiDependencies{
		Unmarshal:           json.Unmarshal,
		Conf:                func() structs.SysConfig { return structs.SysConfig{} },
		StopContainerByName: func(string) error { t.Fatal("stop should not run for unknown action"); return nil },
		StartContainerByName: func(_, _ string) (structs.ContainerState, error) {
			t.Fatal("start should not run for unknown action")
			return structs.ContainerState{}, nil
		},
		UpdateContainerState: func(string, structs.ContainerState) {},
		UpdateConfTyped:      func(...config.ConfUpdateOption) error { return nil },
		WithPenpaiRunning:    config.WithPenpaiRunning,
		WithPenpaiActive:     config.WithPenpaiActive,
		WithPenpaiCores:      config.WithPenpaiCores,
		DeleteContainer:      func(string) error { t.Fatal("delete should not run for unknown action"); return nil },
		NumCPU:               func() int { return 4 },
	}

	if err := HandlePenpai([]byte(`{"payload":{"action":"remove"}}`), deps); err != nil {
		t.Fatalf("remove should be no-op: %v", err)
	}

	if err := HandlePenpai([]byte(`{"payload":{"action":"unknown"}}`), deps); err == nil {
		t.Fatal("expected unknown action to fail")
	}
}
