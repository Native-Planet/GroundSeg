package shipworkflow

import (
	"fmt"
	"strings"

	"groundseg/click"
	"groundseg/docker"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/structs"

	"go.uber.org/zap"
)

func runTogglePowerTransition(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return fmt.Errorf("Failed to get ship status for %s: %w", patp, err)
	}
	status, exists := statuses[patp]
	if !exists {
		return fmt.Errorf("Failed to get ship status for %s: %w", patp, errShipStatusNotFound)
	}
	isRunning := strings.Contains(status, "Up")

	if shipConf.BootStatus == "noboot" {
		return runShipConfigTransition(
			patp,
			"toggle power",
			func() error {
				return persistShipUrbitSectionConfig[structs.UrbitRuntimeConfig](patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "boot"
					return nil
				})
			},
			func(err error) error {
				return fmt.Errorf("Couldn't update urbit config: %w", err)
			},
			func(err error) error {
				return fmt.Errorf("Failed to clean urbit state while toggling power mode for %s: %w", patp, err)
			},
			func(err error) error {
				return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
			},
			shipConfigTransitionStrategy{},
		)
	}
	if shipConf.BootStatus == "boot" && isRunning {
		return runShipConfigTransition(
			patp,
			"toggle power",
			func() error {
				return persistShipUrbitSectionConfig[structs.UrbitRuntimeConfig](patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					conf.BootStatus = "noboot"
					return nil
				})
			},
			func(err error) error {
				return fmt.Errorf("Couldn't update urbit config: %w", err)
			},
			func(err error) error {
				return fmt.Errorf("Failed to clean urbit state while toggling power mode for %s: %w", patp, err)
			},
			nil,
			shipConfigTransitionStrategy{
				shouldCleanup: func(structs.UrbitDocker) bool { return false },
				shouldRestart: func(structs.UrbitDocker) bool { return false },
				postAction: func() error {
					return stopShipAfterPowerToggle(patp)
				},
			},
		)
	}
	if shipConf.BootStatus == "boot" && !isRunning {
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
		}
	}
	return nil
}

func stopShipAfterPowerToggle(patp string) error {
	// set DesiredStatus before stopping to prevent auto-restart from die/stop event handlers
	if containerState, exists := getContainerStatesFn()[patp]; exists {
		containerState.DesiredStatus = "stopped"
		updateContainerStateFn(patp, containerState)
	}
	if err := click.BarExit(patp); err != nil {
		if stopErr := docker.StopContainerByName(patp); stopErr != nil {
			return fmt.Errorf("failed to stop %s: %w", patp, stopErr)
		}
		return fmt.Errorf("failed to stop %s with |exit: %w", patp, err)
	}
	return nil
}

func runToggleNetworkTransition(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	currentNetwork := shipConf.Network
	settings := getStartramSettingsSnapshot()
	zap.L().Warn(fmt.Sprintf("%v", currentNetwork))

	switch currentNetwork {
	case "wireguard":
		return runShipNetworkTransition(patp, "bridge")
	case "bridge":
		if !settings.WgRegistered {
			return fmt.Errorf("No remote registration")
		}
		return runShipNetworkTransition(patp, "wireguard")
	default:
		if !settings.WgRegistered {
			return fmt.Errorf("No remote registration")
		}
		return runShipNetworkTransition(patp, "wireguard")
	}
}

func runShipNetworkTransition(patp string, target string) error {
	return runShipConfigTransition(
		patp,
		"toggle network",
		func() error {
			return persistShipUrbitSectionConfig[structs.UrbitNetworkConfig](patp, dockerOrchestration.UrbitConfigSectionNetwork, func(conf *structs.UrbitNetworkConfig) error {
				conf.Network = target
				return nil
			})
		},
		func(err error) error {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		},
		func(err error) error {
			return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
		},
		func(err error) error {
			return fmt.Errorf("Couldn't start %v: %w", patp, err)
		},
		shipConfigTransitionStrategy{},
	)
}
