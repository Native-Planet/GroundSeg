package lifecycle

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMutateFunctionRequired  = errors.New("mutate function is required")
	ErrPersistFunctionRequired = errors.New("persist function is required")
	ErrPollIntervalNonPositive = errors.New("poll interval must be > 0")
)

// PersistWithMutator validates callback dependencies and executes a typed
// mutation persistence callback.
func PersistWithMutator[T any](id string, mutate func(*T) error, persistFn func(string, func(*T) error) error) error {
	if mutate == nil {
		return ErrMutateFunctionRequired
	}
	if persistFn == nil {
		return ErrPersistFunctionRequired
	}
	return persistFn(id, mutate)
}

// PollWithTimeout polls condition at interval until it returns done=true or
// context cancellation/error.
func PollWithTimeout(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	if interval <= 0 {
		return ErrPollIntervalNonPositive
	}
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
