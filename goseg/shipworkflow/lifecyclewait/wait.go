package lifecyclewait

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

const DefaultStopTimeout = 10 * time.Minute

var (
	ErrShipStatusNotFound = errors.New("ship status not found")
)

type ShipStatusLookup func([]string) (map[string]string, error)
type PollerFunc func(context.Context, time.Duration, func() (bool, error)) error

func WaitForUrbitStopWith(patp string, getStatus ShipStatusLookup, poller PollerFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	const maxStatusFailures = 5
	statusFailureCount := 0
	err := poller(ctx, 500*time.Millisecond, func() (bool, error) {
		statuses, err := getStatus([]string{patp})
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
			missingErr := fmt.Errorf("%w: %s", ErrShipStatusNotFound, patp)
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

func WaitForUrbitStop(patp string, getStatus ShipStatusLookup, poller PollerFunc) error {
	return WaitForUrbitStopWith(patp, getStatus, poller, DefaultStopTimeout)
}
