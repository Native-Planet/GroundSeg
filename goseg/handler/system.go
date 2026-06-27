package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"groundseg/system"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
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
	case "groundseg":
		zap.L().Info(fmt.Sprintf("Device shutdown requested"))
		switch systemPayload.Payload.Command {
		case "restart":
			if err = config.UpdateConf(map[string]any{
				"gracefulExit": true,
			}); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping GroundSeg restart. Exiting program."))
				os.Exit(0)
			} else {
				zap.L().Info(fmt.Sprintf("Restarting GroundSeg.."))
				cmd := exec.Command("systemctl", "restart", "groundseg")
				cmd.Run()
			}
		default:
			return fmt.Errorf("Unrecognized groundseg.service command: %v", systemPayload.Payload.Command)
		}
	case "power":
		switch systemPayload.Payload.Command {
		case "shutdown":
			zap.L().Info(fmt.Sprintf("Device shutdown requested"))
			if err = config.UpdateConf(map[string]any{
				"gracefulExit": true,
			}); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping shutdown. Exiting program."))
				os.Exit(0)
			} else {
				zap.L().Info(fmt.Sprintf("Turning off device.."))
				cmd := exec.Command("shutdown", "-h", "now")
				cmd.Run()
			}
		case "restart":
			zap.L().Info(fmt.Sprintf("Device restart requested"))
			if err = config.UpdateConf(map[string]any{
				"gracefulExit": true,
			}); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if config.DebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping restart. Exiting program."))
				os.Exit(0)
			} else {
				zap.L().Info(fmt.Sprintf("Restarting device.."))
				cmd := exec.Command("reboot")
				cmd.Run()
			}
		default:
			return fmt.Errorf("Unrecognized power command: %v", systemPayload.Payload.Command)
		}
	case "modify-swap":
		zap.L().Info(fmt.Sprintf("Updating swap with value %v", systemPayload.Payload.Value))
		//broadcast.SysTransBus <- structs.SystemTransition{Swap: true, Type: "swap"}
		conf := config.Conf()
		file := conf.SwapFile
		if err := system.ConfigureSwap(file, systemPayload.Payload.Value); err != nil {
			zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
			//broadcast.SysTransBus <- structs.SystemTransition{Swap: false, Type: "swap"}
			return fmt.Errorf("Unable to set swap: %v", err)
		}
		if err = config.UpdateConf(map[string]any{
			"swapVal": systemPayload.Payload.Value,
		}); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't update swap value: %v", err))
		}
		go func() {
			time.Sleep(2 * time.Second)
			//broadcast.SysTransBus <- structs.SystemTransition{Swap: false, Type: "swap"}
		}()
		zap.L().Info(fmt.Sprintf("Swap successfully set to %v", systemPayload.Payload.Value))
	case "update":
		if systemPayload.Payload.Update == "linux" {
			if err := system.RunUpgrade(); err != nil {
				zap.L().Error(fmt.Sprintf("Error updating host system: %v", err))
			}
		}
	case "wifi-toggle":
		if err := system.ToggleDevice(system.Device); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't toggle wifi device: %v", err))
		}
	case "wifi-connect":
		docker.SysTransBus <- structs.SystemTransition{Type: "wifiConnect", Event: "connecting"}
		if err := system.ConnectToWifi(systemPayload.Payload.SSID, systemPayload.Payload.Password); err != nil {
			docker.SysTransBus <- structs.SystemTransition{Type: "wifiConnect", Event: "error"}
			time.Sleep(3 * time.Second)
			docker.SysTransBus <- structs.SystemTransition{Type: "wifiConnect", Event: ""}
			return fmt.Errorf("Couldn't connect to wifi: %v", err)
		}
		docker.SysTransBus <- structs.SystemTransition{Type: "wifiConnect", Event: "success"}
		time.Sleep(3 * time.Second)
		docker.SysTransBus <- structs.SystemTransition{Type: "wifiConnect", Event: ""}
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
}
