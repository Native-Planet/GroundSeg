package provisioning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"groundseg/shipcleanup"
	"groundseg/structs"

	"go.uber.org/zap"
)

type Runtime struct {
	CreateUrbitConfigFn     func(string, string) error
	AppendSysConfigPierFn   func(string) error
	DeleteContainerFn       func(string) error
	DeleteVolumeFn          func(string) error
	CreateVolumeFn          func(string) error
	WriteFileToVolumeFn     func(string, string, string) error
	PublishTransitionFn     func(context.Context, structs.NewShipTransition)
	StartContainerFn        func(string, string) (structs.ContainerState, error)
	UpdateContainerStateFn  func(string, structs.ContainerState)
	ConfigFn                func() structs.SysConfig
	RegisterShipServicesFn  func(string)
	StopContainerByNameFn   func(string) error
	StartLlamaFn            func()
	StartLlamaAPIFn         func()
	WaitForBootCodeFn       func(string, time.Duration)
	WaitForRemoteReadyFn    func(string, time.Duration)
	SwitchShipToWireguardFn func(string, bool) error
	SyncRetrieveFn          func() error
	RollbackProvisioningFn  func(string, shipcleanup.RollbackOptions) error
	SleepFn                 func(time.Duration)
}

func ProvisionShip(runtime Runtime, patp string, shipPayload structs.WsNewShipPayload, customDrive string) error {
	if err := runtime.CreateUrbitConfigFn(patp, customDrive); err != nil {
		errmsg := fmt.Sprintf("failed to create urbit config: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
	}
	if err := runtime.AppendSysConfigPierFn(patp); err != nil {
		errmsg := fmt.Sprintf("failed to add ship to system.json: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
	}

	zap.L().Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	if err := runtime.DeleteContainerFn(patp); err != nil {
		zap.L().Error(fmt.Sprintf("delete container error: %v", err))
	}
	if err := runtime.DeleteVolumeFn(patp); err != nil {
		zap.L().Error(fmt.Sprintf("delete volume error: %v", err))
	}

	if customDrive == "" {
		if err := runtime.CreateVolumeFn(patp); err != nil {
			errmsg := fmt.Sprintf("create volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
		}
		key := shipPayload.Payload.Key
		if err := runtime.WriteFileToVolumeFn(patp, patp+".key", key); err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
		}
	} else {
		path := filepath.Join(customDrive, patp)
		filename := patp + ".key"
		key := shipPayload.Payload.Key
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
		}
		filePath := path + "/" + filename
		if err := os.WriteFile(filePath, []byte(key), 0644); err != nil {
			errmsg := fmt.Sprintf("Error writing to file: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
		}
	}

	zap.L().Info(fmt.Sprintf("Creating Pier: %v", patp))
	runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: "creating"})
	info, err := runtime.StartContainerFn(patp, "vere")
	if err != nil {
		errmsg := fmt.Sprintf("start container error: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
	}
	runtime.UpdateContainerStateFn(patp, info)

	conf := runtime.ConfigFn()
	if conf.Connectivity.WgRegistered {
		go runtime.RegisterShipServicesFn(patp)
	}
	if conf.Penpai.PenpaiAllow {
		if err := runtime.StopContainerByNameFn("llama"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't stop Llama: %v", err))
		}
		runtime.StartLlamaFn()
	}

	go waitForNewShipReady(runtime, shipPayload, customDrive)
	return nil
}

func waitForNewShipReady(runtime Runtime, shipPayload structs.WsNewShipPayload, customDrive string) {
	patp := shipPayload.Payload.Patp
	remote := shipPayload.Payload.Remote

	runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: "booting"})
	zap.L().Info(fmt.Sprintf("Booting ship: %v", patp))
	runtime.WaitForBootCodeFn(patp, 1*time.Second)

	conf := runtime.ConfigFn()
	if conf.Connectivity.WgRegistered && conf.Connectivity.WgOn && remote {
		runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: "remote"})
		runtime.WaitForRemoteReadyFn(patp, 1*time.Second)
		if err := runtime.SwitchShipToWireguardFn(patp, false); err != nil {
			errmsg := fmt.Sprintf("%v", err)
			zap.L().Error(errmsg)
			handleNewShipErrorCleanup(runtime, patp, errmsg, customDrive)
			return
		}
	}

	if err := runtime.SyncRetrieveFn(); err != nil {
		zap.L().Warn(fmt.Sprintf("StarTram sync after ship provisioning failed for %s: %v", patp, err))
	}
	runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: "completed"})
	if conf.Penpai.PenpaiAllow {
		runtime.StartLlamaAPIFn()
	}
}

func handleNewShipErrorCleanup(runtime Runtime, patp, errmsg, customDrive string) error {
	runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: "aborted"})
	runtime.PublishTransitionFn(context.Background(), structs.NewShipTransition{Type: "error", Event: fmt.Sprintf("%v", errmsg)})
	zap.L().Info(fmt.Sprintf("New ship creation failed: %s: %s", patp, errmsg))
	zap.L().Info("Running cleanup routine")
	customPierPath := ""
	if customDrive != "" {
		customPierPath = filepath.Join(customDrive, patp)
	}
	if err := runtime.RollbackProvisioningFn(patp, shipcleanup.RollbackOptions{
		CustomPierPath:       customPierPath,
		RemoveContainer:      true,
		RemoveContainerState: true,
	}); err != nil {
		zap.L().Error(fmt.Sprintf("New ship rollback encountered errors: %v", err))
	}
	return fmt.Errorf("%s", errmsg)
}
