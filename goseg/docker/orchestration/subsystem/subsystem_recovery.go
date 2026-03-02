package subsystem

import (
	"groundseg/logger"
	"groundseg/structs"
	"groundseg/transition"
)

func defaultStopTransitionRestart(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	if state.DesiredStatus == string(transition.ContainerStatusStopped) {
		return nil
	}
	_, err := rt.transitionOps.StartContainerFn(contName, state.Type)
	return err
}

func defaultRestartAfterDeath(rt dockerRoutineRuntime, containerName, containerType string) error {
	go func(name, ctype string) {
		rt.timer.sleepFn(rt.recovery.restartDelay)
		_, err := rt.transitionOps.StartContainerFn(name, ctype)
		if err != nil {
			logger.Errorf("Failed to restart %s after death: %v", name, err)
			return
		}
		logger.Infof("Successfully restarted %s after death", name)
	}(containerName, containerType)
	return nil
}

func defaultRecoverWireguardAfter502(rt dockerRoutineRuntime, settings dockerCheck502Settings) error {
	if settings.Disable502 {
		return nil
	}
	return rt.wireguardOps.recoverWireguardFn(settings.Piers, false)
}
