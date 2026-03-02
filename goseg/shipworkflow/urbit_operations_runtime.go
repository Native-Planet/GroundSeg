package shipworkflow

import (
	"fmt"
	"strings"

	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var runTransitionedOperationFn = RunTransitionedOperation

func toggleAlias(patp string) error {
	currentConf := getUrbitConfigFn(patp)
	nextShowUrbitWeb := "custom"
	if currentConf.ShowUrbitWeb == "custom" {
		nextShowUrbitWeb = "default"
	}
	if err := persistShipWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
		conf.ShowUrbitWeb = nextShowUrbitWeb
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func setUrbitDomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionUrbitDomain, urbitPayload)
}

func setMinIODomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionMinIODomain, urbitPayload)
}

func toggleChopOnVereUpdate(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionChopOnUpgrade, structs.WsUrbitPayload{})
}

func togglePower(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionTogglePower, structs.WsUrbitPayload{})
}

func toggleDevMode(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionToggleDevMode, structs.WsUrbitPayload{})
}

func rebuildContainer(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionRebuildContainer, structs.WsUrbitPayload{})
}

func toggleNetwork(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionToggleNetwork, structs.WsUrbitPayload{})
}

func toggleBootStatus(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	nextBootStatus := "ignore"
	if shipConf.BootStatus == "ignore" {
		statusMap, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			return fmt.Errorf("failed to get ship status for %s: %w", patp, err)
		}
		status, exists := statusMap[patp]
		if !exists {
			return fmt.Errorf("running status for %s doesn't exist: %w", patp, errShipStatusNotFound)
		}
		if strings.Contains(status, "Up") {
			nextBootStatus = "boot"
		} else {
			nextBootStatus = "noboot"
		}
	}
	if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
		conf.BootStatus = nextBootStatus
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func toggleAutoReboot(patp string) error {
	if err := loadUrbitConfigFn(patp); err != nil {
		return fmt.Errorf("Failed to load fresh urbit config: %w", err)
	}
	currentConf := getUrbitConfigFn(patp)
	if err := persistShipFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
		conf.DisableShipRestarts = !currentConf.DisableShipRestarts
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	broadcast.BroadcastToClients()
	return nil
}

func toggleMinIOLink(patp string) error {
	return runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionToggleMinIOLink, structs.WsUrbitPayload{})
}

func buildUrbitDomainSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	currentConf := getUrbitConfigFn(patp)
	return []transitionStep[string]{
		{
			Event: "success",
			Run: func() error {
				alias := payload.Payload.Domain
				oldDomain := currentConf.WgURL
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check Urbit domain alias for %s: %w", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("Invalid Urbit domain alias for %s", patp)
				}
				if err := startram.AliasCreate(patp, alias); err != nil {
					return fmt.Errorf("set urbit domain alias for %s: %w", patp, err)
				}
				if err := persistShipWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
					conf.CustomUrbitWeb = alias
					conf.ShowUrbitWeb = "custom"
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	}
}

func buildMinIODomainSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	currentConf := getUrbitConfigFn(patp)
	return []transitionStep[string]{
		{
			Event: "success",
			Run: func() error {
				alias := payload.Payload.Domain
				oldDomain := fmt.Sprintf("s3.%s", currentConf.WgURL)
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check MinIO domain alias for %s: %w", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("Invalid MinIO domain alias for %s", patp)
				}
				if err := startram.AliasCreate(fmt.Sprintf("s3.%s", patp), alias); err != nil {
					return fmt.Errorf("set minio domain alias for %s: %w", patp, err)
				}
				if err := persistShipWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
					conf.CustomS3Web = alias
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	}
}

func buildRebuildContainerSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	shipConf := getUrbitConfigFn(patp)
	return []transitionStep[string]{
		{
			Run: func() error {
				if err := urbitCleanDelete(patp); err != nil {
					return fmt.Errorf("Failed to clean urbit state for rebuild container transition on %s: %w", patp, err)
				}
				if shipConf.BootStatus != "noboot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
					}
					return nil
				}
				if _, err := docker.CreateContainer(patp, "vere"); err != nil {
					return fmt.Errorf("Failed to create for rebuild container %s: %w", patp, err)
				}
				return nil
			},
		},
	}
}

func buildToggleMinIOLinkSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	var isLinked bool
	var endpoint string
	return []transitionStep[string]{
		{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				// todo: scry for actual info
				isLinked = shipConf.MinIOLinked
				endpoint = shipConf.CustomS3Web
				if endpoint == "" {
					endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
				}
				return nil
			},
		},
		{
			Event: "unlinking",
			EmitWhen: func() bool {
				return isLinked
			},
			Run: func() error {
				if err := click.UnlinkStorage(patp); err != nil {
					return fmt.Errorf("Failed to unlink MinIO information %s: %w", patp, err)
				}

				// Update config
				if err := persistShipFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
					conf.MinIOLinked = false
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
		{
			Event: "unlink-success",
			EmitWhen: func() bool {
				return isLinked
			},
			Run: func() error {
				return nil
			},
		},
		{
			Event: "linking",
			EmitWhen: func() bool {
				return !isLinked
			},
			Run: func() error {
				// create service account
				svcAccount, err := docker.CreateMinIOServiceAccount(patp)
				if err != nil {
					return fmt.Errorf("Failed to create MinIO service account for %s: %w", patp, err)
				}

				// link to urbit
				if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
					return fmt.Errorf("Failed to link MinIO information %s: %w", patp, err)
				}

				// Update config
				if err := persistShipFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
					conf.MinIOLinked = true
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
		{
			Event: "success",
			EmitWhen: func() bool {
				return !isLinked
			},
			Run: func() error {
				return nil
			},
		},
	}
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionLoom, urbitPayload)
	if err != nil {
		return fmt.Errorf("Failed to handle loom transition for %s: %w", patp, err)
	}
	return nil
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommandRegistry(patp, transition.UrbitTransitionSnapTime, urbitPayload)
	if err != nil {
		return fmt.Errorf("Failed to handle snap time transition for %s: %w", patp, err)
	}
	return nil
}

func buildToggleChopOnVereUpdateSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
				currentConf := getUrbitConfigFn(patp)
				if err := persistShipFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
					conf.ChopOnUpgrade = !currentConf.ChopOnUpgrade
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	}
}

func buildTogglePowerSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
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
					if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
						conf.BootStatus = "boot"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					_, err := docker.StartContainer(patp, "vere")
					if err != nil {
						return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
					}
				} else if shipConf.BootStatus == "boot" && isRunning {
					// set DesiredStatus before stopping to prevent auto-restart from die/stop event handlers
					if containerState, exists := getContainerStatesFn()[patp]; exists {
						containerState.DesiredStatus = "stopped"
						updateContainerStateFn(patp, containerState)
					}
					if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
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
				} else if shipConf.BootStatus == "boot" && !isRunning {
					_, err := docker.StartContainer(patp, "vere")
					if err != nil {
						return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
					}
				}
				return nil
			},
		},
	}
}

func buildToggleDevModeSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
				currentConf := getUrbitConfigFn(patp)
				if err := persistShipFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
					conf.DevMode = !currentConf.DevMode
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					return fmt.Errorf("Failed to clean urbit state for dev mode toggle on %s: %w", patp, err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return fmt.Errorf("Couldn't start %v: %w", patp, err)
				}
				return nil
			},
		},
	}
}

func buildToggleNetworkSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				currentNetwork := shipConf.Network
				settings := getStartramSettingsSnapshot()
				zap.L().Warn(fmt.Sprintf("%v", currentNetwork))
				if currentNetwork == "wireguard" {
					if err := persistShipNetworkConfig(patp, func(conf *structs.UrbitNetworkConfig) error {
						conf.Network = "bridge"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					if err := urbitCleanDelete(patp); err != nil {
						return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
					}
				} else if currentNetwork != "wireguard" && settings.WgRegistered {
					if err := persistShipNetworkConfig(patp, func(conf *structs.UrbitNetworkConfig) error {
						conf.Network = "wireguard"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					if err := urbitCleanDelete(patp); err != nil {
						return fmt.Errorf("Failed to clean urbit state while toggling network mode for %s: %w", patp, err)
					}
				} else {
					return fmt.Errorf("No remote registration")
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
					}
				}
				return nil
			},
		},
	}
}

func buildHandleLoomSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
					conf.LoomSize = payload.Payload.Value
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					return fmt.Errorf("Failed to clean urbit state for loom size transition on %s: %w", patp, err)
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
					}
				}
				return nil
			},
		},
	}
}

func buildHandleSnapTimeSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
					conf.SnapTime = payload.Payload.Value
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					return fmt.Errorf("Failed to clean urbit state for snap time transition on %s: %w", patp, err)
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
					}
				}
				return nil
			},
		},
	}
}

func schedulePack(patp string, urbitPayload structs.WsUrbitPayload) error {
	frequency := urbitPayload.Payload.Frequency
	// frequency not 0
	if frequency < 1 {
		return fmt.Errorf("pack frequency cannot be 0!")
	}
	intervalType := urbitPayload.Payload.IntervalType
	switch intervalType {
	case "month", "week", "day":
		if err := persistShipScheduleConfig(patp, func(conf *structs.UrbitScheduleConfig) error {
			conf.MeldTime = urbitPayload.Payload.Time
			conf.MeldSchedule = true
			conf.MeldScheduleType = intervalType
			conf.MeldFrequency = frequency
			conf.MeldDay = urbitPayload.Payload.Day
			conf.MeldDate = urbitPayload.Payload.Date
			return nil
		}); err != nil {
			return fmt.Errorf("Failed to update pack schedule: %w", err)
		}
	default:
		return fmt.Errorf("Schedule pack unknown interval type: %s", intervalType)
	}
	broadcast.PublishSchedulePack("schedule")
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipScheduleConfig(patp, func(conf *structs.UrbitScheduleConfig) error {
		conf.MeldSchedule = false
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to pause pack schedule: %w", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
		conf.SizeLimit = urbitPayload.Payload.Value
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to set new size limit for %s: %w", patp, err)
	}
	return nil
}
