package workflow

import (
	"errors"
	"testing"
	"time"
)

func TestAsyncRunHandleFinishRecordsErrorAndClosesDone(t *testing.T) {
	handle := NewAsyncRunHandle()
	if handle == nil {
		t.Fatal("expected async run handle")
	}

	testErr := errors.New("worker failed")
	handle.Finish(testErr)

	select {
	case <-handle.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected done channel to close after finish")
	}
	if !errors.Is(handle.Err(), testErr) {
		t.Fatalf("expected terminal error %v, got %v", testErr, handle.Err())
	}
}

func TestNewCompletedAsyncRunHandleIsImmediatelyDone(t *testing.T) {
	testErr := errors.New("done")
	handle := NewCompletedAsyncRunHandle(testErr)
	select {
	case <-handle.Done():
	default:
		t.Fatal("expected completed handle to be done")
	}
	if !errors.Is(handle.Err(), testErr) {
		t.Fatalf("expected completed error %v, got %v", testErr, handle.Err())
	}
}
