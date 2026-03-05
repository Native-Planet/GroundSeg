package resource

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

func TestJoinCloseErrorHandlesNilCloserAndSuccess(t *testing.T) {
	baseErr := errors.New("base")
	if got := JoinCloseError(baseErr, nil, "close resource"); !errors.Is(got, baseErr) {
		t.Fatalf("expected base error when closer is nil, got %v", got)
	}
	if got := JoinCloseError(nil, testCloser{}, "close resource"); got != nil {
		t.Fatalf("expected nil error when close succeeds, got %v", got)
	}
}

func TestJoinCloseErrorWrapsAndJoins(t *testing.T) {
	closeErr := errors.New("close failure")
	got := JoinCloseError(nil, testCloser{err: closeErr}, "close resource")
	if got == nil || !strings.Contains(got.Error(), "close resource") {
		t.Fatalf("expected wrapped close error, got %v", got)
	}

	baseErr := errors.New("base")
	got = JoinCloseError(baseErr, testCloser{err: closeErr}, "close resource")
	if !errors.Is(got, baseErr) || !strings.Contains(got.Error(), "close resource") {
		t.Fatalf("expected joined error to include base and close errors, got %v", got)
	}
}
