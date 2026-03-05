package workflow

import (
	"errors"
	"testing"
	"time"
)

type routineLoopStop struct{}

func TestRunForeverCallsFnAndErrorHandler(t *testing.T) {
	expectedErr := errors.New("boom")
	passCount := 0
	errCount := 0

	defer func() {
		if recovered := recover(); recovered != (routineLoopStop{}) {
			t.Fatalf("unexpected panic: %#v", recovered)
		}
		if passCount != 2 {
			t.Fatalf("expected 2 passes, got %d", passCount)
		}
		if errCount != 1 {
			t.Fatalf("expected 1 error callback, got %d", errCount)
		}
	}()

	RunForever(
		time.Millisecond,
		func() error {
			passCount++
			if passCount == 1 {
				return expectedErr
			}
			return nil
		},
		func(time.Duration) {
			if passCount >= 2 {
				panic(routineLoopStop{})
			}
		},
		func(err error) {
			if !errors.Is(err, expectedErr) {
				t.Fatalf("unexpected error callback: %v", err)
			}
			errCount++
		},
	)
}

func TestRunForeverReturnsWhenFnNil(t *testing.T) {
	called := false
	RunForever(time.Millisecond, nil, nil, func(error) {
		called = true
	})
	if called {
		t.Fatal("expected no error callback when function is nil")
	}
}
