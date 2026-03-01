package docker

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultShipExitTimeout = 5 * time.Minute
)

var (
	shipExitPollInterval = 1 * time.Second
	getShipStatusForWait = GetShipStatus
	nowForShipExit       = time.Now
	sleepForShipExit     = time.Sleep
)

// WaitForShipExit polls docker status until the ship is no longer running.
func WaitForShipExit(patp string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultShipExitTimeout
	}

	deadline := nowForShipExit().Add(timeout)
	for {
		statuses, err := getShipStatusForWait([]string{patp})
		if err != nil {
			return fmt.Errorf("failed to get statuses for %s: %v", patp, err)
		}

		status, exists := statuses[patp]
		if !exists {
			return fmt.Errorf("%s status doesn't exist", patp)
		}

		if !strings.Contains(status, "Up") {
			return nil
		}

		if nowForShipExit().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to exit (status=%q)", patp, status)
		}

		sleepForShipExit(shipExitPollInterval)
	}
}
