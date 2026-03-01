package chopsvc

import (
	"context"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	publishUrbitTransitionFn    = docker.PublishUrbitTransition
	getShipStatusFn             = docker.GetShipStatus
	barExitFn                   = click.BarExit
	stopContainerByNameFn       = docker.StopContainerByName
	updateUrbitFn               = config.UpdateUrbit
	startContainerFn            = docker.StartContainer
	forceUpdateContainerStatsFn = docker.ForceUpdateContainerStats
	waitCompleteFn              = WaitComplete
	sleepFn                     = time.Sleep
	waitCompletePollerFn        = pollWithTimeout
)

func ChopPier(patp string, shipConf structs.UrbitDocker) error {
	zap.L().Info(fmt.Sprintf("Chop called for %s", patp))
	chopError := func(err error) error {
		publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: "error"})
		return err
	}
	defer func() {
		sleepFn(3 * time.Second)
		publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: ""})
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
		publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: "stopping"})
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

	publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: "chopping"})
	zap.L().Info(fmt.Sprintf("Attempting to chop %s", patp))
	shipConf.BootStatus = "chop"
	if err := persistShipConf(patp, shipConf); err != nil {
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
		publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: "starting"})
		shipConf.BootStatus = "boot"
		if err := persistShipConf(patp, shipConf); err != nil {
			return chopError(fmt.Errorf("Failed to update %s urbit config to boot: %w", patp, err))
		}
		if _, err := startContainerFn(patp, "vere"); err != nil {
			return chopError(fmt.Errorf("Failed to restart %s after chop: %w", patp, err))
		}
	}
	forceUpdateContainerStatsFn(patp)
	publishUrbitTransitionFn(structs.UrbitTransition{Patp: patp, Type: "chop", Event: "success"})
	return nil
}

func WaitComplete(patp string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	const maxStatusFailures = 5
	statusFailureCount := 0
	err := waitCompletePollerFn(ctx, 500*time.Millisecond, func() (bool, error) {
		statuses, err := getShipStatusFn([]string{patp})
		if err != nil {
			statusFailureCount++
			wrappedErr := fmt.Errorf("retrieve ship status for %s: %w", patp, err)
			if statusFailureCount >= maxStatusFailures {
				return false, wrappedErr
			}
			zap.L().Warn(fmt.Sprintf("Retrying wait-complete status retrieval for %s (%d/%d): %v", patp, statusFailureCount, maxStatusFailures, wrappedErr))
			return false, nil
		}
		status, exists := statuses[patp]
		if !exists {
			statusFailureCount++
			missingErr := fmt.Errorf("status for %s not found", patp)
			if statusFailureCount >= maxStatusFailures {
				return false, missingErr
			}
			zap.L().Warn(fmt.Sprintf("Retrying wait-complete status lookup for %s (%d/%d): %v", patp, statusFailureCount, maxStatusFailures, missingErr))
			return false, nil
		}
		statusFailureCount = 0
		if strings.Contains(status, "Up") {
			zap.L().Debug(fmt.Sprintf("%s continue waiting...", patp))
			return false, nil
		}
		zap.L().Debug(fmt.Sprintf("%s finished", patp))
		return true, nil
	})
	if err == context.DeadlineExceeded {
		zap.L().Warn(fmt.Sprintf("%s timed out waiting for completion", patp))
		return err
	}
	if err != nil {
		zap.L().Error(fmt.Sprintf("%s wait-complete failed: %v", patp, err))
		return err
	}
	return nil
}

func persistShipConf(patp string, shipConf structs.UrbitDocker) error {
	return updateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		*conf = shipConf
		return nil
	})
}

func pollWithTimeout(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		done, err := condition()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
