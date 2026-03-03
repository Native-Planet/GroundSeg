package system

import (
	"context"
	"encoding/json"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/handler/systemsvc"
	"groundseg/structs"
	"groundseg/system"
	"os/exec"
	"time"
)

var (
	confForSystemHandler             = config.Conf
	stopContainerForSystemHandler    = orchestration.StopContainerByName
	updateConfTypedForSystemHandler  = config.UpdateConfTyped
	withPenpaiAllowForSystemHandler  = config.WithPenpaiAllow
	loadLlamaForSystemHandler        = orchestration.LoadLlama
	withGracefulExitForSystemHandler = config.WithGracefulExit
	execCommandForSystemHandler      = func(name string, args ...string) systemsvc.CommandRunner {
		return exec.Command(name, args...)
	}
	configureSwapForSystemHandler = system.ConfigureSwap
	withSwapValForSystemHandler   = config.WithSwapVal
	runUpgradeForSystemHandler    = system.RunUpgrade
	toggleDeviceForSystemHandler  = func(dev string) error {
		return system.DefaultWiFiRuntimeService().ToggleDevice(dev)
	}
	connectToWifiForSystemHandler = func(ssid, password string) error {
		return system.DefaultWiFiRuntimeService().ConnectToWifi(ssid, password)
	}
	publishSystemTransitionForSystemHandler = func(_ context.Context, transition structs.SystemTransition) error {
		_ = events.DefaultEventRuntime().PublishSystemTransition(context.Background(), transition)
		return nil
	}
	sleepForSystemHandler = time.Sleep
)

// handle system events
func SystemHandler(msg []byte) error {
	return systemsvc.HandleSystem(msg, systemsvc.SystemDependencies{
		Unmarshal:           json.Unmarshal,
		Conf:                confForSystemHandler,
		StopContainerByName: stopContainerForSystemHandler,
		UpdateConfTyped:     updateConfTypedForSystemHandler,
		WithPenpaiAllow:     withPenpaiAllowForSystemHandler,
		LoadLlama:           loadLlamaForSystemHandler,
		WithGracefulExit:    withGracefulExitForSystemHandler,
		ExecCommand:         execCommandForSystemHandler,
		ConfigureSwap:       configureSwapForSystemHandler,
		WithSwapVal:         withSwapValForSystemHandler,
		RunUpgrade:          runUpgradeForSystemHandler,
		ToggleDevice:        toggleDeviceForSystemHandler,
		ConnectToWifi:       connectToWifiForSystemHandler,
		PublishSystemTransition: func(transition structs.SystemTransition) {
			_ = publishSystemTransitionForSystemHandler(context.Background(), transition)
		},
		Sleep:       sleepForSystemHandler,
		IsDebugMode: config.DebugMode(),
	})
}
