package shipworkflow

import (
	"fmt"
	"groundseg/click"
	"groundseg/docker"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/structs"
	"groundseg/transition"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

var runUrbitTransitionTemplateFn = runUrbitTransitionTemplate
var sleepFn = time.Sleep

func wrapLifecycleError(patp string, detail string, err error) error {
	return fmt.Errorf("%s for %s: %w", detail, patp, err)
}

func packPier(patp string) error {
	return runPackLifecycle(patp)
}

func RunPack(patp string) error {
	return runPackLifecycle(patp)
}

func RunScheduledPack(patp string, delay time.Duration) error {
	if delay > 0 {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s in %v", patp, delay))
		sleepFn(delay)
	} else {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s", patp))
	}
	return runPackLifecycle(patp)
}

func runPackLifecycle(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionPack, structs.WsUrbitPayload{})
}

func packMeldPier(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionPackMeld, structs.WsUrbitPayload{})
}

func rollChopPier(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionRollChop, structs.WsUrbitPayload{})
}

func buildPackSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	packError := func(err error) error {
		return fmt.Errorf("pack operation: %w", err)
	}
	return []transitionStep[string]{
		{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return packError(wrapLifecycleError(patp, "Failed to get ship status", err))
				}
				status, exists := statuses[patp]
				if !exists {
					return packError(fmt.Errorf("Failed to get ship status for %s: %w", patp, errShipStatusNotFound))
				}
				// running
				if strings.Contains(status, "Up") {
					// send |pack
					if err := click.SendPack(patp); err != nil {
						return packError(wrapLifecycleError(patp, "Failed to send pack command", err))
					}
					// not running
				} else {
					// set DesiredStatus to prevent auto-restart when pack container exits
					if containerState, exists := getContainerStatesFn()[patp]; exists {
						containerState.DesiredStatus = "stopped"
						updateContainerStateFn(patp, containerState)
					}
					// switch boot status to pack
					err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
						conf.BootStatus = "pack"
						return nil
					})
					if err != nil {
						return packError(wrapLifecycleError(patp, "Failed to update urbit config to pack", err))
					}
				}
				// set last meld
				now := time.Now().Unix()
				err = persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionSchedule, func(conf *structs.UrbitScheduleConfig) error {
					conf.MeldLast = strconv.FormatInt(now, 10)
					return nil
				})
				if err != nil {
					return packError(wrapLifecycleError(patp, "Failed to update urbit config with last meld time", err))
				}
				return nil
			},
		},
	}
}

func buildPackMeldSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	var isRunning bool
	return []transitionStep[string]{
		{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return wrapLifecycleError(patp, "Failed to get ship status", err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: %w", patp, errShipStatusNotFound)
				}
				isRunning = strings.Contains(status, "Up")
				// set DesiredStatus to prevent auto-restart from die/stop event handlers during maintenance
				if containerState, exists := getContainerStatesFn()[patp]; exists {
					containerState.DesiredStatus = "stopped"
					updateContainerStateFn(patp, containerState)
				}
				return nil
			},
		},
		{
			Event: "stopping",
			Run: func() error {
				if !isRunning {
					return nil
				}
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for pack & meld %s: %v", patp, err))
					if err = docker.StopContainerByName(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to stop ship for pack & meld %s: %v", patp, err))
					}
				}
				if err := WaitComplete(patp); err != nil {
					return wrapLifecycleError(patp, "Failed waiting for stop completion on %s before pack & meld", err)
				}
				return nil
			},
		},
		{
			Run: func() error {
				// start ship as pack
				zap.L().Info(fmt.Sprintf("Attempting to urth pack %s", patp))
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "pack"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to pack", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start pack container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for urth pack to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for pack completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		{
			Event: "melding",
			Run: func() error {
				// start ship as meld
				zap.L().Info(fmt.Sprintf("Attempting to urth meld %s", patp))
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "meld"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to meld", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start meld container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for urth meld to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for meld completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		{
			Event: "starting",
			Run: func() error {
				if !isRunning {
					return nil
				}
				// restore DesiredStatus so normal auto-restart behavior resumes
				if containerState, exists := getContainerStatesFn()[patp]; exists {
					containerState.DesiredStatus = "running"
					updateContainerStateFn(patp, containerState)
				}
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "boot"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to meld", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start meld container", err)
				}
				return nil
			},
		},
	}
}

func buildRollChopSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	var isRunning bool
	return []transitionStep[string]{
		{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return wrapLifecycleError(patp, "Failed to get ship status", err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: %w", patp, errShipStatusNotFound)
				}
				isRunning = strings.Contains(status, "Up")
				return nil
			},
		},
		{
			Event: "stopping",
			EmitWhen: func() bool {
				return isRunning
			},
			Run: func() error {
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for roll & chop %s: %v", patp, err))
					if err = docker.StopContainerByName(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to stop ship for roll & chop %s: %v", patp, err))
					}
				}
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for stop completion on %s before roll & chop: %w", patp, err)
				}
				return nil
			},
		},
		{
			Run: func() error {
				// start ship as roll
				zap.L().Info(fmt.Sprintf("Attempting to roll %s", patp))
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "roll"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to roll", err)
				}
				if _, err := docker.StartContainer(patp, "vere"); err != nil {
					return fmt.Errorf("Failed to start roll container %s: %w", patp, err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for roll to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for roll completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		{
			Event: "chopping",
			Run: func() error {
				// start ship as chop
				zap.L().Info(fmt.Sprintf("Attempting to chop %s", patp))
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "chop"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to chop", err)
				}
				if _, err := docker.StartContainer(patp, "vere"); err != nil {
					return fmt.Errorf("Failed to start chop container %s: %w", patp, err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for chop completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		{
			Event: "starting",
			EmitWhen: func() bool {
				return isRunning
			},
			Run: func() error {
				if err := persistShipUrbitConfig(patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "boot"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to chop", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return fmt.Errorf("Failed to start chop container %s: %w", patp, err)
				}
				return nil
			},
		},
	}
}
