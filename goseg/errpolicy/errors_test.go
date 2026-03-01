package errpolicy

import (
	"errors"
	"testing"
)

func TestWrapOperationWrapsWithCause(t *testing.T) {
	cause := errors.New("boom")
	err := WrapOperation("upload op", cause)
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if err.Error() != "upload op: boom" {
		t.Fatalf("unexpected wrapped error string: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped error to preserve cause chain")
	}
}

func TestWrapOperationNilPassthrough(t *testing.T) {
	if err := WrapOperation("noop", nil); err != nil {
		t.Fatalf("expected nil passthrough, got %v", err)
	}
}
