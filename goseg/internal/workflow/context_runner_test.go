package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunUntilDoneOrWorkerResultReturnsWorkerError(t *testing.T) {
	want := errors.New("boom")
	err := RunUntilDoneOrWorkerResult(context.Background(),
		func(context.Context) error { return want },
		func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected worker error %v, got %v", want, err)
	}
}

func TestRunUntilDoneOrWorkerResultReturnsNilOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunUntilDoneOrWorkerResult(ctx, func(context.Context) error { return nil }); err != nil {
		t.Fatalf("expected nil for canceled context, got %v", err)
	}
}

func TestRunUntilDoneOrWorkerResultHandlesNoWorkers(t *testing.T) {
	if err := RunUntilDoneOrWorkerResult(context.Background()); err != nil {
		t.Fatalf("expected nil with no workers, got %v", err)
	}
}

func TestRunUntilDoneOrWorkerResultReturnsNilWorkerResult(t *testing.T) {
	err := RunUntilDoneOrWorkerResult(context.Background(),
		func(context.Context) error { return nil },
		func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(200 * time.Millisecond):
				return errors.New("expected cancellation")
			}
		},
	)
	if err != nil {
		t.Fatalf("expected first worker nil result, got %v", err)
	}
}
