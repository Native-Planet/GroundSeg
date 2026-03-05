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
	maintenanceapt "groundseg/system/maintenance/apt"
	"groundseg/transition"
	"os/exec"
	"time"
)

var (
	confForSystemHandler             = config.Config
	stopContainerForSystemHandler    = orchestration.StopContainerByName
	updateConfTypedForSystemHandler  = config.UpdateConfigTyped
	withPenpaiAllowForSystemHandler  = config.WithPenpaiAllow
	loadLlamaForSystemHandler        = orchestration.LoadLlama
	withGracefulExitForSystemHandler = config.WithGracefulExit
	execCommandForSystemHandler      = func(name string, args ...string) systemsvc.CommandRunner {
		return exec.Command(name, args...)
	}
	configureSwapForSystemHandler = system.ConfigureSwap
	withSwapValForSystemHandler   = config.WithSwapVal
	runUpgradeForSystemHandler    = maintenanceapt.RunUpgrade
	toggleDeviceForSystemHandler  = func(dev string) error {
		return system.NewWiFiRuntimeService().ToggleDevice(dev)
	}
	connectToWifiForSystemHandler = func(ssid, password string) error {
		return system.NewWiFiRuntimeService().ConnectToWiFi(ssid, password)
	}
	publishSystemTransitionForSystemHandler = func(ctx context.Context, transition structs.SystemTransition) error {
		return events.DefaultEventRuntime().PublishSystemTransition(ctx, transition)
	}
	sleepForSystemHandler = time.Sleep
)

// handle system events
func SystemHandler(msg []byte) error {
	return systemsvc.HandleSystem(msg, systemsvc.SystemDependencies{
		Unmarshal:           json.Unmarshal,
		Config:              confForSystemHandler,
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
		ConnectToWiFi:       connectToWifiForSystemHandler,
		PublishSystemTransition: func(ctx context.Context, transition structs.SystemTransition) error {
			return publishSystemTransitionForSystemHandler(ctx, transition)
		},
		TransitionPublishPolicy: transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		Sleep:                   sleepForSystemHandler,
		IsDebugMode:             config.DebugMode(),
	})
}
