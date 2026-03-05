package shipworkflow

import (
	"fmt"

	"groundseg/docker"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/structs"
)

func buildSingleStepTransition(patp string, runFn func() error) []transitionStep[string] {
	_ = patp
	return []transitionStep[string]{
		{
			Run: runFn,
		},
	}
}

func buildRebuildContainerSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		if err := urbitCleanDelete(patp); err != nil {
			return fmt.Errorf("failed to clean urbit state for rebuild container transition on %s: %w", patp, err)
		}
		shipConf := getUrbitConfigFn(patp)
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

func buildToggleChopOnVereUpdateSteps(patp string, _ structs.WsUrbitPayload) []transitionStep[string] {
	return buildSingleStepTransition(patp, func() error {
		currentConf := getUrbitConfigFn(patp)
		return persistShipUrbitSectionConfig[structs.UrbitFeatureConfig](patp, dockerOrchestration.UrbitConfigSectionFeature, func(conf *structs.UrbitFeatureConfig) error {
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
		return runShipConfigTransition(
			patp,
			"toggle dev mode",
			func() error {
				currentConf := getUrbitConfigFn(patp)
				return persistShipUrbitSectionConfig[structs.UrbitFeatureConfig](patp, dockerOrchestration.UrbitConfigSectionFeature, func(conf *structs.UrbitFeatureConfig) error {
					conf.DevMode = !currentConf.DevMode
					return nil
				})
			},
			func(err error) error {
				return fmt.Errorf("could not update urbit config: %w", err)
			},
			func(err error) error {
				return fmt.Errorf("failed to clean urbit state for dev mode toggle on %s: %w", patp, err)
			},
			func(err error) error {
				return fmt.Errorf("could not start %v: %w", patp, err)
			},
			shipConfigTransitionStrategy{},
		)
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
	return buildSingleStepTransition(patp, func() error {
		return runShipConfigTransition(
			patp,
			op,
			func() error {
				return persistShipUrbitSectionConfig[structs.UrbitRuntimeConfig](patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
					setter(conf)
					return nil
				})
			},
			func(err error) error {
				return fmt.Errorf("could not update urbit config: %w", err)
			},
			func(err error) error {
				return fmt.Errorf("failed to clean urbit state for %s transition on %s: %w", op, patp, err)
			},
			func(err error) error {
				return fmt.Errorf("could not start %v: %w", patp, err)
			},
			shipConfigTransitionStrategy{},
		)
	})
}

type shipConfigTransitionStrategy struct {
	shouldCleanup func(structs.UrbitDocker) bool
	shouldRestart func(structs.UrbitDocker) bool
	postAction    func() error
}

func runShipConfigTransition(
	patp string,
	operation string,
	updateConfig func() error,
	onPersistErr func(error) error,
	onCleanupErr func(error) error,
	onStartErr func(error) error,
	strategy shipConfigTransitionStrategy,
) error {
	if err := updateConfig(); err != nil {
		if onPersistErr != nil {
			return onPersistErr(err)
		}
		return fmt.Errorf("could not update config for %s: %w", operation, err)
	}
	shipConf := getUrbitConfigFn(patp)
	shouldCleanup := true
	if strategy.shouldCleanup != nil {
		shouldCleanup = strategy.shouldCleanup(shipConf)
	}
	if shouldCleanup {
		if err := urbitCleanDelete(patp); err != nil {
			if onCleanupErr != nil {
				return onCleanupErr(err)
			}
			return fmt.Errorf("failed to clean urbit state for %s transition on %s: %w", operation, patp, err)
		}
	}
	shouldRestart := shipConf.BootStatus == "boot"
	if strategy.shouldRestart != nil {
		shouldRestart = strategy.shouldRestart(shipConf)
	}
	if shouldRestart {
		if _, err := docker.StartContainer(patp, "vere"); err != nil {
			if onStartErr != nil {
				return onStartErr(err)
			}
			return fmt.Errorf("failed to restart %v after %s transition: %w", patp, operation, err)
		}
	}
	if strategy.postAction != nil {
		if err := strategy.postAction(); err != nil {
			return err
		}
	}
	return nil
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
