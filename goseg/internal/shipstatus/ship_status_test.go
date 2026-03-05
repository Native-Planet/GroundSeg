package shipstatus

import (
	"errors"
	"testing"
)

func TestNotFoundErrWrapsSentinelAndIncludesPatp(t *testing.T) {
	err := NotFoundErr("~zod")
	if !errors.Is(err, ErrShipStatusNotFound) {
		t.Fatalf("expected NotFoundErr to wrap ErrShipStatusNotFound, got %v", err)
	}
	if err.Error() == ErrShipStatusNotFound.Error() {
		t.Fatalf("expected NotFoundErr to include ship identifier, got %q", err.Error())
	}
}
