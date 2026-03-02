package shipworkflow

import (
	"fmt"
	"time"

	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/shipcleanup"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

func deleteShip(patp string) error {
	settings := getStartramSettingsSnapshot()
	removeServices := false
	return runUrbitTransitionTemplate(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionDeleteShip),
		startEvent:     "stopping",
		successEvent:   "done",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     1 * time.Second,
	},
		transitionStep[string]{
			Run: func() error {
				// update DesiredStatus to 'stopped'
				contConf := getContainerStatesFn()
				patpConf := contConf[patp]
				patpConf.DesiredStatus = "stopped"
				contConf[patp] = patpConf
				updateContainerStateFn(patp, patpConf)
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					if err := docker.StopContainerByName(patp); err != nil {
						return fmt.Errorf("Couldn't stop docker container for %v: %v", patp, err)
					}
				}
				if err := docker.DeleteContainer(patp); err != nil {
					return fmt.Errorf("Couldn't delete docker container for %v: %v", patp, err)
				}
				removeServices = settings.WgRegistered
				return nil
			},
		},
		transitionStep[string]{
			Event: "removing-services",
			EmitWhen: func() bool {
				return removeServices
			},
			Run: func() error {
				if err := startram.SvcDelete(patp, "urbit"); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v: %v", patp, err))
				}
				if err := startram.SvcDelete("s3."+patp, "s3"); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v: %v", patp, err))
				}
				if err := docker.DeleteContainer("minio_" + patp); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't delete minio docker container for %v: %v", patp, err))
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "deleting",
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				customPath := shipConf.CustomPierLocation
				if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
					CustomPierPath:       customPath,
					RemoveContainerState: true,
				}); err != nil {
					zap.L().Error(fmt.Sprintf("Ship cleanup encountered errors for %v: %v", patp, err))
				}
				zap.L().Info(fmt.Sprintf("%v container deleted", patp))
				// remove from broadcast
				if err := broadcast.ReloadUrbits(); err != nil {
					zap.L().Error(fmt.Sprintf("Error updating broadcast: %v", err))
				}
				return nil
			},
		},
	)
}

func exportShip(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionTemplate(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionExportShip),
		startEvent:     "stopping",
		successEvent:   "ready",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     1 * time.Second,
	},
		transitionStep[string]{
			Run: func() error {
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "noboot"
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				// stop container
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					if err := docker.StopContainerByName(patp); err != nil {
						return err
					}
				}
				// whitelist the patp token pair
				if err := exporter.WhitelistContainer(patp, urbitPayload.Token); err != nil {
					return err
				}
				return nil
			},
		},
	)
}

func exportBucket(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionTemplate(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionExportBucket),
		successEvent:   "ready",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     1 * time.Second,
		err: func(error) string {
			return "error"
		},
	},
		transitionStep[string]{
			Run: func() error {
				containerName := fmt.Sprintf("minio_%s", patp)
				// whitelist the patp token pair
				if err := exporter.WhitelistContainer(containerName, urbitPayload.Token); err != nil {
					return err
				}
				return nil
			},
		},
	)
}
