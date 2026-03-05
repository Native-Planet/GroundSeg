package lifecyclewait

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fixedAttemptPoller(maxAttempts int) PollerFunc {
	return func(ctx context.Context, _ time.Duration, condition func() (bool, error)) error {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
		if ctx == nil {
			return nil
		}
		return ctx.Err()
	}
}

func TestWaitForUrbitStopWithReturnsWhenShipStops(t *testing.T) {
	err := WaitForUrbitStopWith(
		"~zod",
		func([]string) (map[string]string, error) {
			return map[string]string{"~zod": "Exited (0) 1 second ago"}, nil
		},
		fixedAttemptPoller(1),
		time.Second,
	)
	if err != nil {
		t.Fatalf("WaitForUrbitStopWith returned error: %v", err)
	}
}

func TestWaitForUrbitStopWithFailsAfterStatusErrors(t *testing.T) {
	statusErr := errors.New("status unavailable")
	err := WaitForUrbitStopWith(
		"~zod",
		func([]string) (map[string]string, error) {
			return nil, statusErr
		},
		fixedAttemptPoller(5),
		time.Second,
	)
	if err == nil {
		t.Fatal("expected WaitForUrbitStopWith to fail after repeated status errors")
	}
	if !errors.Is(err, statusErr) {
		t.Fatalf("expected status error in chain, got %v", err)
	}
}

func TestWaitForUrbitStopWithFailsWhenShipStatusMissing(t *testing.T) {
	err := WaitForUrbitStopWith(
		"~zod",
		func([]string) (map[string]string, error) {
			return map[string]string{}, nil
		},
		fixedAttemptPoller(5),
		time.Second,
	)
	if err == nil {
		t.Fatal("expected WaitForUrbitStopWith to fail when ship status is missing")
	}
	if !errors.Is(err, ErrShipStatusNotFound) {
		t.Fatalf("expected ErrShipStatusNotFound, got %v", err)
	}
}
