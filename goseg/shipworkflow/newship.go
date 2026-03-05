package shipworkflow

import (
	"context"
	"fmt"
	"time"

	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/network"
	"groundseg/docker/orchestration"
	"groundseg/shipcleanup"
	"groundseg/shipcreator"
	"groundseg/shipworkflow/provisioning"
	"groundseg/startram"
	"groundseg/structs"

	"go.uber.org/zap"
)

func newShipProvisioningRuntime() provisioning.Runtime {
	networkRuntime := network.NewNetworkRuntime()
	return provisioning.Runtime{
		CreateUrbitConfigFn:   shipcreator.CreateUrbitConfig,
		AppendSysConfigPierFn: shipcreator.AppendSysConfigPier,
		DeleteContainerFn:     orchestration.DeleteContainer,
		DeleteVolumeFn:        networkRuntime.DeleteVolume,
		CreateVolumeFn:        networkRuntime.CreateVolume,
		WriteFileToVolumeFn:   networkRuntime.WriteFileToVolume,
		PublishTransitionFn: func(ctx context.Context, transition structs.NewShipTransition) {
			events.DefaultEventRuntime().PublishNewShipTransition(ctx, transition)
		},
		StartContainerFn:       orchestration.StartContainer,
		UpdateContainerStateFn: config.UpdateContainerState,
		ConfigFn:               config.Config,
		RegisterShipServicesFn: RegisterNewShipServices,
		StopContainerByNameFn:  orchestration.StopContainerByName,
		StartLlamaFn: func() {
			if _, err := orchestration.StartContainer("llama", "llama"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't restart Llama: %v", err))
			}
		},
		StartLlamaAPIFn: func() {
			if _, err := orchestration.StartContainer("llama-gpt-api", "llama-api"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't start Llama API: %v", err))
			}
		},
		WaitForBootCodeFn:       WaitForBootCode,
		WaitForRemoteReadyFn:    WaitForRemoteReady,
		SwitchShipToWireguardFn: SwitchShipToWireguard,
		SyncRetrieveFn: func() error {
			_, err := startram.SyncRetrieve()
			return err
		},
		RollbackProvisioningFn: shipcleanup.RollbackProvisioning,
		SleepFn:                time.Sleep,
	}
}

func ProvisionShip(patp string, shipPayload structs.WsNewShipPayload, customDrive string) error {
	return provisioning.ProvisionShip(newShipProvisioningRuntime(), patp, shipPayload, customDrive)
}

func RegisterNewShipServices(patp string) {
	if err := RegisterShipServices(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
