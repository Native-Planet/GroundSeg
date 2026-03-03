package chopsvc

import (
	"context"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	publishUrbitTransitionFn = func(ctx context.Context, transition structs.UrbitTransition) error {
		return events.DefaultEventRuntime().PublishUrbitTransition(ctx, transition)
	}
	getShipStatusFn             = orchestration.GetShipStatus
	barExitFn                   = click.BarExit
	stopContainerByNameFn       = orchestration.StopContainerByName
	startContainerFn            = orchestration.StartContainer
	forceUpdateContainerStatsFn = orchestration.ForceUpdateContainerStats
	persistShipRuntimeConfigFn  = config.UpdateUrbitRuntimeConfig
	waitCompleteFn              = func(patp string) error {
		return WaitComplete(patp)
	}
	sleepFn              = time.Sleep
	waitCompletePollerFn = shipworkflow.PollWithTimeout
)

func ChopPier(patp string) error {
	zap.L().Info(fmt.Sprintf("Chop called for %s", patp))
	publishTransition := func(event string) {
		publishUrbitTransitionFn(context.Background(), structs.UrbitTransition{Patp: patp, Type: "chop", Event: event})
	}
	chopError := func(err error) error {
		publishTransition("error")
		return err
	}
	defer func() {
		sleepFn(3 * time.Second)
		publishTransition("")
		zap.L().Info(fmt.Sprintf("Chop for %s, ran defer", patp))
	}()

	statuses, err := getShipStatusFn([]string{patp})
	if err != nil {
		return chopError(fmt.Errorf("Failed to get ship status for %s: %w", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return chopError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	if isRunning {
		publishTransition("stopping")
		if err := barExitFn(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for chop %s: %v", patp, err))
			if err = stopContainerByNameFn(patp); err != nil {
				return fmt.Errorf("Failed to stop ship for chop %s: %w", patp, err)
			}
		}
		if err := waitCompleteFn(patp); err != nil {
			return chopError(fmt.Errorf("Failed waiting for stop completion on %s before chop: %w", patp, err))
		}
	}

	publishTransition("chopping")
	zap.L().Info(fmt.Sprintf("Attempting to chop %s", patp))
	if err := persistShipRuntimeConfigFn(patp, func(conf *structs.UrbitRuntimeConfig) error {
		conf.BootStatus = "chop"
		return nil
	}); err != nil {
		return chopError(fmt.Errorf("Failed to update %s urbit config to chop: %w", patp, err))
	}
	if _, err := startContainerFn(patp, "vere"); err != nil {
		return chopError(fmt.Errorf("Failed to chop %s: %w", patp, err))
	}

	zap.L().Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
	if err := waitCompleteFn(patp); err != nil {
		return chopError(fmt.Errorf("Failed waiting for chop completion on %s: %w", patp, err))
	}

	if isRunning {
		publishTransition("starting")
		if err := persistShipRuntimeConfigFn(patp, func(conf *structs.UrbitRuntimeConfig) error {
			conf.BootStatus = "boot"
			return nil
		}); err != nil {
			return chopError(fmt.Errorf("Failed to update %s urbit config to boot: %w", patp, err))
		}
		if _, err := startContainerFn(patp, "vere"); err != nil {
			return chopError(fmt.Errorf("Failed to restart %s after chop: %w", patp, err))
		}
	}
	forceUpdateContainerStatsFn(patp)
	publishTransition("success")
	return nil
}

func WaitComplete(patp string) error {
	return shipworkflow.WaitForUrbitStop(patp, getShipStatusFn, waitCompletePollerFn)
}
