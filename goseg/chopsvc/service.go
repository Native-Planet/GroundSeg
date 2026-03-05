package chopsvc

import (
	"context"
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/shipworkflow/adapters/lifecyclebridge"
	"groundseg/shipworkflow/lifecyclewait"
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
	persistShipRuntimeConfigFn  = func(patp string, update func(*structs.UrbitRuntimeConfig) error) error {
		return config.UpdateUrbitRuntimeConfig(patp, update)
	}
	waitCompleteFn = func(patp string) error {
		return WaitComplete(patp)
	}
	sleepFn              = time.Sleep
	waitCompletePollerFn = lifecyclebridge.PollWithTimeout
)

func ChopPier(patp string) error {
	zap.L().Info(fmt.Sprintf("Chop called for %s", patp))
	publishTransition := func(event string) error {
		return publishUrbitTransitionFn(context.Background(), structs.UrbitTransition{Patp: patp, Type: "chop", Event: event})
	}
	chopError := func(err error) error {
		publishErr := publishTransition("error")
		if publishErr == nil {
			return err
		}
		return errors.Join(publishErr, err)
	}
	defer func() {
		sleepFn(3 * time.Second)
		if err := publishTransition(""); err != nil {
			zap.L().Warn(fmt.Sprintf("failed to publish chop transition completion: %v", err))
		}
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
		if err := publishTransition("stopping"); err != nil {
			return err
		}
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

	if err := publishTransition("chopping"); err != nil {
		return chopError(fmt.Errorf("Failed to publish chop transition for %s: %w", patp, err))
	}
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
		if err := publishTransition("starting"); err != nil {
			return chopError(fmt.Errorf("Failed to publish restart transition for %s: %w", patp, err))
		}
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
	if err := publishTransition("success"); err != nil {
		return err
	}
	return nil
}

func WaitComplete(patp string) error {
	return lifecyclewait.WaitForUrbitStop(patp, getShipStatusFn, waitCompletePollerFn)
}
