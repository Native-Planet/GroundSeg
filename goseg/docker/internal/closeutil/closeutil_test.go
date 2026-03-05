package closeutil

import (
	"errors"
	"strings"
	"testing"
)

type testCloser struct {
	err error
}

func (closer testCloser) Close() error {
	return closer.err
}

func TestMergeCloseErrorNoopPaths(t *testing.T) {
	var opErr error
	MergeCloseError(nil, "noop", &opErr)
	if opErr != nil {
		t.Fatalf("expected nil error when closer is nil")
	}
	MergeCloseError(testCloser{}, "noop", nil)
}

func TestMergeCloseErrorWrapsAndJoins(t *testing.T) {
	closeErr := errors.New("close failure")
	var opErr error
	MergeCloseError(testCloser{err: closeErr}, "shutdown", &opErr)
	if opErr == nil || !strings.Contains(opErr.Error(), "close docker client during shutdown") {
		t.Fatalf("expected wrapped close error, got %v", opErr)
	}

	baseErr := errors.New("base")
	opErr = baseErr
	MergeCloseError(testCloser{err: closeErr}, "shutdown", &opErr)
	if !errors.Is(opErr, baseErr) || !strings.Contains(opErr.Error(), "close docker client during shutdown") {
		t.Fatalf("expected joined error to include both base and close errors: %v", opErr)
	}
}
