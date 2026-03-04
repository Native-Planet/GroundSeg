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
)

var runTransitionedOperationFn = RunTransitionedOperation

var urbitTransitionRunners = UrbitTransitionRunners()

func runUrbitTransitionFromCommand(patp string, transitionType transition.UrbitTransitionType, payload structs.WsUrbitPayload) error {
	runFn, ok := urbitTransitionRunners[transitionType]
	if !ok {
		return runUrbitTransitionFromCommandRegistry(patp, transitionType, payload)
	}
	return runFn(patp, payload)
}

func toggleAlias(patp string) error {
	currentConf := getUrbitConfigFn(patp)
	nextShowUrbitWeb := "custom"
	if currentConf.ShowUrbitWeb == "custom" {
		nextShowUrbitWeb = "default"
	}
	if err := persistShipUrbitWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
		conf.ShowUrbitWeb = nextShowUrbitWeb
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func setUrbitDomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionUrbitDomain, urbitPayload)
}

func setMinIODomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionMinIODomain, urbitPayload)
}

func toggleChopOnVereUpdate(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionChopOnUpgrade, structs.WsUrbitPayload{})
}

func togglePower(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionTogglePower, structs.WsUrbitPayload{})
}

func toggleDevMode(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionToggleDevMode, structs.WsUrbitPayload{})
}

func rebuildContainer(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionRebuildContainer, structs.WsUrbitPayload{})
}

func toggleNetwork(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionToggleNetwork, structs.WsUrbitPayload{})
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
	if err := persistShipUrbitRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
		conf.BootStatus = nextBootStatus
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func toggleAutoReboot(patp string) error {
	if err := loadUrbitConfigFn(patp); err != nil {
		return fmt.Errorf("failed to load fresh urbit config: %w", err)
	}
	currentConf := getUrbitConfigFn(patp)
	if err := persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
		conf.DisableShipRestarts = !currentConf.DisableShipRestarts
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	broadcast.BroadcastToClients()
	return nil
}

func toggleMinIOLink(patp string) error {
	return runUrbitTransitionFromCommand(patp, transition.UrbitTransitionToggleMinIOLink, structs.WsUrbitPayload{})
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
					return fmt.Errorf("failed to check urbit domain alias for %s: %w", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("invalid urbit domain alias for %s", patp)
				}
				if err := startram.AliasCreate(patp, alias); err != nil {
					return fmt.Errorf("set urbit domain alias for %s: %w", patp, err)
				}
				if err := persistShipUrbitWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
					conf.CustomUrbitWeb = alias
					conf.ShowUrbitWeb = "custom"
					return nil
				}); err != nil {
					return fmt.Errorf("could not update urbit config: %w", err)
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
					return fmt.Errorf("failed to check minio domain alias for %s: %w", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("invalid minio domain alias for %s", patp)
				}
				if err := startram.AliasCreate(fmt.Sprintf("s3.%s", patp), alias); err != nil {
					return fmt.Errorf("set minio domain alias for %s: %w", patp, err)
				}
				if err := persistShipUrbitWebConfig(patp, func(conf *structs.UrbitWebConfig) error {
					conf.CustomS3Web = alias
					return nil
				}); err != nil {
					return fmt.Errorf("could not update urbit config: %w", err)
				}
				return nil
			},
		},
	}
}

func buildSingleStepTransition(patp string, runFn func() error) []transitionStep[string] {
	_ = patp
	return []transitionStep[string]{
		{
			Run: runFn,
		},
	}
}

func buildRebuildContainerSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	shipConf := getUrbitConfigFn(patp)
	return buildSingleStepTransition(patp, func() error {
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("failed to clean urbit state for rebuild container transition on %s: %w", patp, err)
		}
		if shipConf.BootStatus != "noboot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				return fmt.Errorf("failed to start container for rebuild %s: %w", patp, err)
			}
			return nil
		}
		if _, err := docker.CreateContainer(patp, "vere"); err != nil {
			return fmt.Errorf("failed to create container for rebuild %s: %w", patp, err)
		}
		return nil
	})
}

func buildToggleMinIOLinkSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	coordinator := minIOLinkTransitionCoordinator{patp: patp}
	return coordinator.steps()
}

type minIOLinkTransitionCoordinator struct {
	patp      string
	linkState minIOLinkState
}

func (c *minIOLinkTransitionCoordinator) steps() []transitionStep[string] {
	return []transitionStep[string]{
		{
			Run: c.loadCurrentState,
		},
		{
			Event: "unlinking",
			EmitWhen: func() bool {
				return c.linkState.IsLinked
			},
			Run: c.unlinkIfNeeded,
		},
		{
			Event: "unlink-success",
			EmitWhen: func() bool {
				return c.linkState.IsLinked
			},
		},
		{
			Event: "linking",
			EmitWhen: func() bool {
				return !c.linkState.IsLinked
			},
			Run: c.linkIfNeeded,
		},
		{
			Event: "success",
			EmitWhen: func() bool {
				return !c.linkState.IsLinked
			},
		},
	}
}

func (c *minIOLinkTransitionCoordinator) loadCurrentState() error {
	state, stateErr := loadMinIOLinkState(c.patp)
	if stateErr != nil {
		return stateErr
	}
	c.linkState = state
	return nil
}

func (c *minIOLinkTransitionCoordinator) unlinkIfNeeded() error {
	if !c.linkState.IsLinked {
		return nil
	}
	return unbindMinIOLink(c.patp)
}

func (c *minIOLinkTransitionCoordinator) linkIfNeeded() error {
	if c.linkState.IsLinked {
		return nil
	}
	return linkMinIOLink(c.patp, c.linkState)
}

type minIOLinkState struct {
	IsLinked bool
	Endpoint string
}

func loadMinIOLinkState(patp string) (minIOLinkState, error) {
	shipConf := getUrbitConfigFn(patp)
	endpoint := shipConf.CustomS3Web
	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
	}
	return minIOLinkState{
		IsLinked: shipConf.MinIOLinked,
		Endpoint: endpoint,
	}, nil
}

func unbindMinIOLink(patp string) error {
	if err := click.UnlinkStorage(patp); err != nil {
		return fmt.Errorf("failed to unlink minio information %s: %w", patp, err)
	}
	if err := persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
		conf.MinIOLinked = false
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func linkMinIOLink(patp string, state minIOLinkState) error {
	svcAccount, err := docker.CreateMinIOServiceAccount(patp)
	if err != nil {
		return fmt.Errorf("failed to create minio service account for %s: %w", patp, err)
	}
	if err := click.LinkStorage(patp, state.Endpoint, svcAccount); err != nil {
		return fmt.Errorf("failed to link minio information %s: %w", patp, err)
	}
	if err := persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
		conf.MinIOLinked = true
		return nil
	}); err != nil {
		return fmt.Errorf("could not update urbit config: %w", err)
	}
	return nil
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommand(patp, transition.UrbitTransitionLoom, urbitPayload)
	if err != nil {
		return fmt.Errorf("failed to handle loom transition for %s: %w", patp, err)
	}
	return nil
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommand(patp, transition.UrbitTransitionSnapTime, urbitPayload)
	if err != nil {
		return fmt.Errorf("failed to handle snap time transition for %s: %w", patp, err)
	}
	return nil
}

func buildToggleChopOnVereUpdateSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		currentConf := getUrbitConfigFn(patp)
		return persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
			conf.ChopOnUpgrade = !currentConf.ChopOnUpgrade
			return nil
		})
	})
}

func buildTogglePowerSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		return runTogglePowerTransition(patp)
	})
}

func buildToggleDevModeSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		currentConf := getUrbitConfigFn(patp)
		if err := persistShipUrbitFeatureConfig(patp, func(conf *structs.UrbitFeatureConfig) error {
			conf.DevMode = !currentConf.DevMode
			return nil
		}); err != nil {
			return fmt.Errorf("could not update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("failed to clean urbit state for dev mode toggle on %s: %w", patp, err)
		}
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			return fmt.Errorf("could not start %v: %w", patp, err)
		}
		return nil
	})
}

func buildToggleNetworkSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		return runToggleNetworkTransition(patp)
	})
}

func buildRuntimeConfigIntSteps(
	patp string,
	payload structs.WsUrbitPayload,
	setter func(*structs.UrbitRuntimeConfig),
	op string,
) []transitionStep[string] {
	shipConf := getUrbitConfigFn(patp)
	return buildSingleStepTransition(patp, func() error {
		if err := persistShipUrbitRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
			setter(conf)
			return nil
		}); err != nil {
			return fmt.Errorf("could not update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("failed to clean urbit state for %s transition on %s: %w", op, patp, err)
		}
		if shipConf.BootStatus == "boot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				return fmt.Errorf("could not start %v: %w", patp, err)
			}
		}
		return nil
	})
}

func buildHandleLoomSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return buildRuntimeConfigIntSteps(patp, payload, func(conf *structs.UrbitRuntimeConfig) {
		conf.LoomSize = payload.Payload.Value
	}, "loom size")
}

func buildHandleSnapTimeSteps(patp string, payload structs.WsUrbitPayload) []transitionStep[string] {
	return buildRuntimeConfigIntSteps(patp, payload, func(conf *structs.UrbitRuntimeConfig) {
		conf.SnapTime = payload.Payload.Value
	}, "snap time")
}

func schedulePack(patp string, urbitPayload structs.WsUrbitPayload) error {
	frequency := urbitPayload.Payload.Frequency
	// frequency not 0
	if frequency < 1 {
		return fmt.Errorf("pack frequency must be greater than zero")
	}
	intervalType := urbitPayload.Payload.IntervalType
	switch intervalType {
	case "month", "week", "day":
		if err := persistShipUrbitScheduleConfig(patp, func(conf *structs.UrbitScheduleConfig) error {
			conf.MeldTime = urbitPayload.Payload.Time
			conf.MeldSchedule = true
			conf.MeldScheduleType = intervalType
			conf.MeldFrequency = frequency
			conf.MeldDay = urbitPayload.Payload.Day
			conf.MeldDate = urbitPayload.Payload.Date
			return nil
		}); err != nil {
			return fmt.Errorf("failed to update pack schedule: %w", err)
		}
	default:
		return fmt.Errorf("unknown pack schedule interval type: %s", intervalType)
	}
	if err := broadcast.DefaultBroadcastStateRuntime().PublishSchedulePack("schedule"); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("publish schedule-pack transition for %s", patp),
			err,
			transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		)
	}
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipUrbitScheduleConfig(patp, func(conf *structs.UrbitScheduleConfig) error {
		conf.MeldSchedule = false
		return nil
	}); err != nil {
		return fmt.Errorf("failed to pause pack schedule: %w", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipUrbitRuntimeConfig(patp, func(conf *structs.UrbitRuntimeConfig) error {
		conf.SizeLimit = urbitPayload.Payload.Value
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set new size limit for %s: %w", patp, err)
	}
	return nil
}
