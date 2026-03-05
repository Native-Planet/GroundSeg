package subsystem

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"groundseg/config"
	"groundseg/internal/shipstatus"
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
	if !sleepOrContextDone(ctx, rt.recovery.check502InitialDelay) {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		if !sleepOrContextDone(ctx, rt.recovery.check502PollDelay) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		settings, threshold, pierStatus, err := check502CycleSnapshot(rt)
		if err != nil {
			logger.Errorf("couldn't get pier status: %v", err)
			continue
		}
		for _, pier := range settings.Piers {
			if !processCheck502Pier(ctx, rt, settings, threshold, pier, pierStatus, status) {
				return
			}
		}
	}
}

func check502CycleSnapshot(rt dockerRoutineRuntime) (config.Check502Settings, int, map[string]string, error) {
	threshold := rt.recovery.check502ConsecutiveFailures
	if threshold < 1 {
		threshold = 1
	}
	settings := rt.Check502SettingsSnapshotFn()
	settings.Piers = append([]string(nil), settings.Piers...)
	pierStatus, err := rt.GetShipStatusFn(settings.Piers)
	if err != nil {
		return settings, threshold, nil, err
	}
	return settings, threshold, pierStatus, nil
}

func processCheck502Pier(
	ctx context.Context,
	rt dockerRoutineRuntime,
	settings config.Check502Settings,
	threshold int,
	pier string,
	pierStatus map[string]string,
	status map[string]int,
) bool {
	if ctx.Err() != nil {
		return false
	}
	if err := rt.LoadUrbitConfigFn(pier); err != nil {
		logger.Errorf("error loading %s config: %v", pier, err)
		return true
	}
	shipConf := rt.UrbitConfFn(pier)
	pierNetwork, err := rt.GetContainerNetworkFn(pier)
	if err != nil {
		logger.Warnf("couldn't get network for %v: %v", pier, err)
		return true
	}
	if !transition.IsContainerUpStatus(pierStatus[pier]) || pierNetwork == "default" || !settings.WgOn {
		return true
	}
	if _, err := rt.GetLusCodeFn(pier); err != nil {
		logger.Warnf("%v is not booted yet, skipping", pier)
		return true
	}
	resp, err := rt.httpOps.getFn("https://" + shipConf.WgURL)
	if err != nil {
		logger.Errorf("error remote polling %v: %v", pier, err)
		return true
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		logger.Errorf("failed to close 502 check response for %v: %v", pier, closeErr)
	}
	logger.Debugf("%v 502 check: %v", pier, resp.StatusCode)
	if resp.StatusCode == http.StatusBadGateway {
		logger.Warnf("Got 502 response for %v", pier)
		status[pier]++
		if status[pier] < threshold {
			return true
		}
		if settings.Disable502 {
			delete(status, pier)
			return true
		}
		logger.Warnf("502 strike %d/%d for %v", status[pier], threshold, pier)
		if err := rt.recovery.recoverWireguardAfter502Fn(rt, settings); err != nil {
			logger.Errorf("wireguard fleet recovery failed: %v", err)
		}
		delete(status, pier)
		return true
	}
	delete(status, pier)
	return true
}

func sleepOrContextDone(ctx context.Context, d time.Duration) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
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
		statuses, err := rt.GetShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", shipstatus.NotFoundErr(patp)
		}
		return status, nil
	}
	settings := rt.ShipSettingsSnapshotFn()
	piers := append([]string(nil), settings.Piers...)
	pierStatus, err := rt.GetShipStatusFn(piers)
	if err != nil {
		return fmt.Errorf("failed to retrieve ship information: %w", err)
	}
	steps := []workflow.Step{}
	for patp, status := range pierStatus {
		if transition.IsContainerUpStatus(status) {
			pirate := patp
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("stop %s with |exit for daemon restart", pirate),
				Run: func() error {
					if err := rt.systemOps.barExitFn(pirate); err != nil {
						return fmt.Errorf("failed to stop %s with |exit for daemon restart: %w", pirate, err)
					}
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
