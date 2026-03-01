package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"groundseg/system"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

var (
	confForSystemHandler                    = config.Conf
	stopContainerForSystemHandler           = docker.StopContainerByName
	updateConfTypedForSystemHandler         = config.UpdateConfTyped
	withPenpaiAllowForSystemHandler         = config.WithPenpaiAllow
	loadLlamaForSystemHandler               = docker.LoadLlama
	withGracefulExitForSystemHandler        = config.WithGracefulExit
	execCommandForSystemHandler             = exec.Command
	configureSwapForSystemHandler           = system.ConfigureSwap
	withSwapValForSystemHandler             = config.WithSwapVal
	runUpgradeForSystemHandler              = system.RunUpgrade
	toggleDeviceForSystemHandler            = system.ToggleDevice
	connectToWifiForSystemHandler           = system.ConnectToWifi
	publishSystemTransitionForSystemHandler = docker.PublishSystemTransition
	sleepForSystemHandler                   = time.Sleep
)

// handle system events
func SystemHandler(msg []byte) error {
	zap.L().Info("System")
	var systemPayload structs.WsSystemPayload
	err := json.Unmarshal(msg, &systemPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal system payload: %v", err)
	}
	switch systemPayload.Payload.Action {
	case "toggle-penpai-feature":
		conf := confForSystemHandler()
		if conf.PenpaiAllow {
			err := stopContainerForSystemHandler("llama-gpt-api")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop Llama API: %v", err))
			}
			err = stopContainerForSystemHandler("llama-gpt-ui")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop Llama UI: %v", err))
			}
			if err = updateConfTypedForSystemHandler(withPenpaiAllowForSystemHandler(false)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't toggle penpai feature: %v", err))
			}
		} else {
			if err = updateConfTypedForSystemHandler(withPenpaiAllowForSystemHandler(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't toggle penpai feature: %v", err))
			}
			if err := loadLlamaForSystemHandler(); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to load llama docker: %v", err))
			}
		}
	case "groundseg":
		zap.L().Info(fmt.Sprintf("Device shutdown requested"))
		switch systemPayload.Payload.Command {
		case "restart":
			if err = updateConfTypedForSystemHandler(withGracefulExitForSystemHandler(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping GroundSeg restart."))
				return nil
			} else {
				zap.L().Info(fmt.Sprintf("Restarting GroundSeg.."))
				cmd := execCommandForSystemHandler("systemctl", "restart", "groundseg")
				cmd.Run()
			}
		default:
			return fmt.Errorf("Unrecognized groundseg.service command: %v", systemPayload.Payload.Command)
		}
	case "power":
		switch systemPayload.Payload.Command {
		case "shutdown":
			zap.L().Info(fmt.Sprintf("Device shutdown requested"))
			if err = updateConfTypedForSystemHandler(withGracefulExitForSystemHandler(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping shutdown."))
				return nil
			} else {
				zap.L().Info(fmt.Sprintf("Turning off device.."))
				cmd := execCommandForSystemHandler("shutdown", "-h", "now")
				cmd.Run()
			}
		case "restart":
			zap.L().Info(fmt.Sprintf("Device restart requested"))
			if err = updateConfTypedForSystemHandler(withGracefulExitForSystemHandler(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping restart."))
				return nil
			} else {
				zap.L().Info(fmt.Sprintf("Restarting device.."))
				cmd := execCommandForSystemHandler("reboot")
				cmd.Run()
			}
		default:
			return fmt.Errorf("Unrecognized power command: %v", systemPayload.Payload.Command)
		}
	case "modify-swap":
		zap.L().Info(fmt.Sprintf("Updating swap with value %v", systemPayload.Payload.Value))
		//broadcast.SysTransBus <- structs.SystemTransition{Swap: true, Type: "swap"}
		conf := confForSystemHandler()
		file := conf.SwapFile
		if err := configureSwapForSystemHandler(file, systemPayload.Payload.Value); err != nil {
			zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
			//broadcast.SysTransBus <- structs.SystemTransition{Swap: false, Type: "swap"}
			return fmt.Errorf("Unable to set swap: %v", err)
		}
		if err = updateConfTypedForSystemHandler(withSwapValForSystemHandler(systemPayload.Payload.Value)); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't update swap value: %v", err))
		}
		go func() {
			sleepForSystemHandler(2 * time.Second)
			//broadcast.SysTransBus <- structs.SystemTransition{Swap: false, Type: "swap"}
		}()
		zap.L().Info(fmt.Sprintf("Swap successfully set to %v", systemPayload.Payload.Value))
	case "update":
		if systemPayload.Payload.Update == "linux" {
			if err := runUpgradeForSystemHandler(); err != nil {
				zap.L().Error(fmt.Sprintf("Error updating host system: %v", err))
			}
		}
	case "wifi-toggle":
		if err := toggleDeviceForSystemHandler(system.Device); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't toggle wifi device: %v", err))
		}
	case "wifi-connect":
		publishSystemTransitionForSystemHandler(structs.SystemTransition{Type: "wifiConnect", Event: "connecting"})
		if err := connectToWifiForSystemHandler(systemPayload.Payload.SSID, systemPayload.Payload.Password); err != nil {
			publishSystemTransitionForSystemHandler(structs.SystemTransition{Type: "wifiConnect", Event: "error"})
			sleepForSystemHandler(3 * time.Second)
			publishSystemTransitionForSystemHandler(structs.SystemTransition{Type: "wifiConnect", Event: ""})
			return fmt.Errorf("Couldn't connect to wifi: %v", err)
		}
		publishSystemTransitionForSystemHandler(structs.SystemTransition{Type: "wifiConnect", Event: "success"})
		sleepForSystemHandler(3 * time.Second)
		publishSystemTransitionForSystemHandler(structs.SystemTransition{Type: "wifiConnect", Event: ""})
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
}
