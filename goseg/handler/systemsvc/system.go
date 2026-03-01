package systemsvc

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"groundseg/system"
	"time"

	"go.uber.org/zap"
)

type CommandRunner interface {
	Run() error
}

type SystemDependencies struct {
	Unmarshal               func(data []byte, target any) error
	Conf                    func() structs.SysConfig
	StopContainerByName     func(name string) error
	UpdateConfTyped         func(...config.ConfUpdateOption) error
	WithPenpaiAllow         func(bool) config.ConfUpdateOption
	LoadLlama               func() error
	WithGracefulExit        func(bool) config.ConfUpdateOption
	ExecCommand             func(string, ...string) CommandRunner
	ConfigureSwap           func(file string, value int) error
	WithSwapVal             func(int) config.ConfUpdateOption
	RunUpgrade              func() error
	ToggleDevice            func(string) error
	ConnectToWifi           func(ssid, password string) error
	PublishSystemTransition func(structs.SystemTransition)
	Sleep                   func(time.Duration)
	IsDebugMode             bool
}

type PenpaiDependencies struct {
	Unmarshal            func(data []byte, target any) error
	Conf                 func() structs.SysConfig
	StopContainerByName  func(name string) error
	StartContainerByName func(name, containerType string) (structs.ContainerState, error)
	UpdateContainerState func(name string, state structs.ContainerState)
	UpdateConfTyped      func(...config.ConfUpdateOption) error
	WithPenpaiRunning    func(bool) config.ConfUpdateOption
	WithPenpaiActive     func(string) config.ConfUpdateOption
	WithPenpaiCores      func(int) config.ConfUpdateOption
	DeleteContainer      func(name string) error
	NumCPU               func() int
}

func HandleSystem(msg []byte, deps SystemDependencies) error {
	var systemPayload structs.WsSystemPayload
	unmarshal := deps.Unmarshal
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	if err := unmarshal(msg, &systemPayload); err != nil {
		return fmt.Errorf("Couldn't unmarshal system payload: %v", err)
	}

	switch systemPayload.Payload.Action {
	case "toggle-penpai-feature":
		conf := deps.Conf()
		if conf.PenpaiAllow {
			if err := deps.StopContainerByName("llama-gpt-api"); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop Llama API: %v", err))
			}
			if err := deps.StopContainerByName("llama-gpt-ui"); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop Llama UI: %v", err))
			}
			if err := deps.UpdateConfTyped(deps.WithPenpaiAllow(false)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't toggle penpai feature: %v", err))
			}
		} else {
			if err := deps.UpdateConfTyped(deps.WithPenpaiAllow(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't toggle penpai feature: %v", err))
			}
			if err := deps.LoadLlama(); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to load llama docker: %v", err))
			}
		}
	case "groundseg":
		zap.L().Info(fmt.Sprintf("Device shutdown requested"))
		switch systemPayload.Payload.Command {
		case "restart":
			if err := deps.UpdateConfTyped(deps.WithGracefulExit(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if deps.IsDebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping GroundSeg restart."))
				return nil
			}
			zap.L().Info(fmt.Sprintf("Restarting GroundSeg.."))
			cmd := deps.ExecCommand("systemctl", "restart", "groundseg")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restart groundseg: %w", err)
			}
		default:
			return fmt.Errorf("Unrecognized groundseg.service command: %v", systemPayload.Payload.Command)
		}
	case "power":
		switch systemPayload.Payload.Command {
		case "shutdown":
			zap.L().Info(fmt.Sprintf("Device shutdown requested"))
			if err := deps.UpdateConfTyped(deps.WithGracefulExit(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if deps.IsDebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping shutdown."))
				return nil
			}
			zap.L().Info(fmt.Sprintf("Turning off device.."))
			cmd := deps.ExecCommand("shutdown", "-h", "now")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to shutdown device: %w", err)
			}
		case "restart":
			zap.L().Info(fmt.Sprintf("Device restart requested"))
			if err := deps.UpdateConfTyped(deps.WithGracefulExit(true)); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't set graceful exit to true: %v", err))
			}
			if deps.IsDebugMode {
				zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping restart."))
				return nil
			}
			zap.L().Info(fmt.Sprintf("Restarting device.."))
			cmd := deps.ExecCommand("reboot")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restart device: %w", err)
			}
		default:
			return fmt.Errorf("Unrecognized power command: %v", systemPayload.Payload.Command)
		}
	case "modify-swap":
		zap.L().Info(fmt.Sprintf("Updating swap with value %v", systemPayload.Payload.Value))
		conf := deps.Conf()
		file := conf.SwapFile
		if err := deps.ConfigureSwap(file, systemPayload.Payload.Value); err != nil {
			zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
			return fmt.Errorf("Unable to set swap: %v", err)
		}
		if err := deps.UpdateConfTyped(deps.WithSwapVal(systemPayload.Payload.Value)); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't update swap value: %v", err))
		}
		go func() {
			deps.Sleep(2 * time.Second)
		}()
		zap.L().Info(fmt.Sprintf("Swap successfully set to %v", systemPayload.Payload.Value))
	case "update":
		if systemPayload.Payload.Update == "linux" {
			if err := deps.RunUpgrade(); err != nil {
				zap.L().Error(fmt.Sprintf("Error updating host system: %v", err))
			}
		}
	case "wifi-toggle":
		if err := deps.ToggleDevice(system.Device); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't toggle wifi device: %v", err))
		}
	case "wifi-connect":
		deps.PublishSystemTransition(structs.SystemTransition{Type: "wifiConnect", Event: "connecting"})
		if err := deps.ConnectToWifi(systemPayload.Payload.SSID, systemPayload.Payload.Password); err != nil {
			deps.PublishSystemTransition(structs.SystemTransition{Type: "wifiConnect", Event: "error"})
			deps.Sleep(3 * time.Second)
			deps.PublishSystemTransition(structs.SystemTransition{Type: "wifiConnect", Event: ""})
			return fmt.Errorf("Couldn't connect to wifi: %v", err)
		}
		deps.PublishSystemTransition(structs.SystemTransition{Type: "wifiConnect", Event: "success"})
		deps.Sleep(3 * time.Second)
		deps.PublishSystemTransition(structs.SystemTransition{Type: "wifiConnect", Event: ""})
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
}

func HandlePenpai(msg []byte, deps PenpaiDependencies) error {
	unmarshal := deps.Unmarshal
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	var penpaiPayload structs.WsPenpaiPayload
	if err := unmarshal(msg, &penpaiPayload); err != nil {
		return fmt.Errorf("Couldn't unmarshal penpai payload: %v", err)
	}
	conf := deps.Conf()
	switch penpaiPayload.Payload.Action {
	case "toggle":
		running := false
		if conf.PenpaiRunning {
			if err := deps.StopContainerByName("llama-gpt-api"); err != nil {
				return fmt.Errorf("Failed to stop Llama API: %v", err)
			}
			if err := deps.StopContainerByName("llama-gpt-ui"); err != nil {
				return fmt.Errorf("Failed to stop Llama UI: %v", err)
			}
		} else {
			info, err := deps.StartContainerByName("llama-gpt-api", "llama-api")
			if err != nil {
				return fmt.Errorf("Error starting Llama API: %v", err)
			}
			deps.UpdateContainerState("llama-api", info)
			running = true
		}
		if err := deps.UpdateConfTyped(deps.WithPenpaiRunning(running)); err != nil {
			return fmt.Errorf("%v", err)
		}
		return nil
	case "set-model":
		model := penpaiPayload.Payload.Model
		if err := deps.UpdateConfTyped(deps.WithPenpaiActive(model)); err != nil {
			return fmt.Errorf("%v", err)
		}
		if err := deps.DeleteContainer("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		if conf.PenpaiRunning {
			if _, err := deps.StartContainerByName("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
	case "set-cores":
		cores := penpaiPayload.Payload.Cores
		if cores < 1 {
			return fmt.Errorf("Penpai unable to set 0 cores!")
		}
		if cores >= deps.NumCPU() {
			return fmt.Errorf("Penpai unable to set %v cores!", cores)
		}
		if err := deps.UpdateConfTyped(deps.WithPenpaiCores(cores)); err != nil {
			return fmt.Errorf("%v", err)
		}
		if err := deps.DeleteContainer("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		if conf.PenpaiRunning {
			if _, err := deps.StartContainerByName("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
	case "remove":
		zap.L().Debug(fmt.Sprintf("Todo: remove penpai"))
	default:
		return fmt.Errorf("Unrecognized penpai action: %v", penpaiPayload.Payload.Action)
	}
	return nil
}
