package subsystem

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"groundseg/internal/workflow"
	"groundseg/logger"
	"groundseg/transition"
)

// loop to make sure ships are reachable
// if 502 2x in 2 min, restart wg container
func Check502Loop(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	check502Loop(ctx, newDockerRoutineRuntime())
}

func check502Loop(ctx context.Context, rt dockerRoutineRuntime) {
	if ctx == nil {
		ctx = context.Background()
	}
	status := make(map[string]int)
	rt.timer.sleepFn(rt.recovery.check502InitialDelay)
	for {
		if ctx.Err() != nil {
			return
		}
		rt.timer.sleepFn(rt.recovery.check502PollDelay)
		if ctx.Err() != nil {
			return
		}
		threshold := rt.recovery.check502ConsecutiveFailures
		if threshold < 1 {
			threshold = 1
		}
		settings := rt.healthOps.Check502SettingsSnapshotFn()
		pierStatus, err := rt.healthOps.GetShipStatusFn(settings.Piers)
		if err != nil {
			logger.Errorf("Couldn't get pier status: %v", err)
			continue
		}
		for _, pier := range settings.Piers {
			if ctx.Err() != nil {
				return
			}
			err := rt.transitionOps.LoadUrbitConfigFn(pier)
			if err != nil {
				logger.Errorf("Error loading %s config: %v", pier, err)
				continue
			}
			shipConf := rt.transitionOps.UrbitConfFn(pier)
			pierNetwork, err := rt.healthOps.GetContainerNetworkFn(pier)
			if err != nil {
				logger.Warnf("Couldn't get network for %v: %v", pier, err)
				continue
			}
			turnedOn := false
			if transition.IsContainerUpStatus(pierStatus[pier]) {
				turnedOn = true
			}
			if turnedOn && pierNetwork != "default" && settings.WgOn {
				if _, err := rt.healthOps.GetLusCodeFn(pier); err != nil {
					logger.Warnf("%v is not booted yet, skipping", pier)
					continue
				}
				resp, err := rt.httpOps.getFn("https://" + shipConf.WgURL)
				if err != nil {
					logger.Errorf("Error remote polling %v: %v", pier, err)
					continue
				}
				resp.Body.Close()
				logger.Debugf("%v 502 check: %v", pier, resp.StatusCode)
				if resp.StatusCode == http.StatusBadGateway {
					logger.Warnf("Got 502 response for %v", pier)
					status[pier]++
					if status[pier] < threshold {
						continue
					}
					if settings.Disable502 {
						delete(status, pier)
						continue
					}
					logger.Warnf("502 strike %d/%d for %v", status[pier], threshold, pier)
					if err := rt.recovery.recoverWireguardAfter502Fn(rt, settings); err != nil {
						logger.Errorf("Wireguard fleet recovery failed: %v", err)
					}
					// remove from map after recovery attempt
					delete(status, pier)
				} else if _, found := status[pier]; found {
					// if not 502 and pier is in status map, remove it
					delete(status, pier)
				}
			}
		}
	}
}

func GracefulShipExit() error {
	return gracefulShipExit(newDockerRoutineRuntime())
}

func gracefulShipExit(rt dockerRoutineRuntime) error {
	DisableShipRestart = true
	defer func() {
		DisableShipRestart = false
	}()
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := rt.healthOps.GetShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	piers := rt.healthOps.ShipSettingsSnapshotFn().Piers
	pierStatus, err := rt.healthOps.GetShipStatusFn(piers)
	if err != nil {
		return fmt.Errorf("Failed to retrieve ship information: %w", err)
	}
	steps := []workflow.Step{}
	for patp, status := range pierStatus {
		if transition.IsContainerUpStatus(status) {
			pirate := patp
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("stop %s with |exit for daemon restart", pirate),
				Run:  func() error { return rt.systemOps.barExitFn(pirate) },
			})
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("wait for %s status during graceful exit", pirate),
				Run: func() error {
					for {
						status, err := getShipRunningStatus(pirate)
						if err != nil {
							return fmt.Errorf("failed to poll %s status during graceful exit: %w", pirate, err)
						}
						logger.Debugf("%s", status)
						if !transition.IsContainerUpStatus(status) {
							return nil
						}
						rt.timer.sleepFn(1 * time.Second)
					}
				},
			})
		}
	}
	if joined := workflow.Join(steps, func(err error) {
		logger.Errorf("one or more ships failed graceful shutdown: %v", err)
	}); joined != nil {
		return fmt.Errorf("one or more ships failed graceful shutdown: %w", joined)
	}
	return nil
}
