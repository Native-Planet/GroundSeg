package shipworkflow

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"groundseg/click"
	"groundseg/docker"
	"groundseg/structs"
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
		if err := persistShipUrbitRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
			conf.BootStatus = "boot"
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
		}
		return nil
	}
	if shipConf.BootStatus == "boot" && isRunning {
		// set DesiredStatus before stopping to prevent auto-restart from die/stop event handlers
		if containerState, exists := getContainerStatesFn()[patp]; exists {
			containerState.DesiredStatus = "stopped"
			updateContainerStateFn(patp, containerState)
		}
		if err := persistShipUrbitRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
			conf.BootStatus = "noboot"
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		err := click.BarExit(patp)
		if err != nil {
			if err := docker.StopContainerByName(patp); err != nil {
				return fmt.Errorf("failed to stop %s: %w", patp, err)
			}
			return fmt.Errorf("failed to stop %s with |exit: %w", patp, err)
		}
		return nil
	}
	if shipConf.BootStatus == "boot" && !isRunning {
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
		}
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
		if err := persistShipUrbitNetworkConfig(patp, func(conf *structs.UrbitNetworkConfig) error {
			conf.Network = "bridge"
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
		}
	case "bridge":
		if !settings.WgRegistered {
			return fmt.Errorf("No remote registration")
		}
		if err := persistShipUrbitNetworkConfig(patp, func(conf *structs.UrbitNetworkConfig) error {
			conf.Network = "wireguard"
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
		}
	default:
		if !settings.WgRegistered {
			return fmt.Errorf("No remote registration")
		}
		if err := persistShipUrbitNetworkConfig(patp, func(conf *structs.UrbitNetworkConfig) error {
			conf.Network = "wireguard"
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
		}
	}

	if shipConf.BootStatus == "boot" {
		if _, err := docker.StartContainer(patp, "vere"); err != nil {
			return fmt.Errorf("Couldn't start %v: %w", patp, err)
		}
	}
	return nil
}
