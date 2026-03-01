package startram

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapAPIConnectionErrorRedactsUpstreamDetailsAndPreservesCause(t *testing.T) {
	cause := errors.New("dial https://api.example/v1/retrieve?pubkey=abc123DEF4560K&token=secret: timeout")
	wrapped := wrapAPIConnectionError(cause)
	if wrapped == nil {
		t.Fatal("expected wrapped error")
	}

	msg := wrapped.Error()
	if msg != apiConnectionErrorMessage {
		t.Fatalf("expected stable redacted message %q, got %q", apiConnectionErrorMessage, msg)
	}
	for _, forbidden := range []string{"abc123DEF456", "pubkey=", "token=secret", "timeout"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("expected redacted outward message to hide %q, got %q", forbidden, msg)
		}
	}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("expected wrapped error to preserve cause chain")
	}
	if errors.Unwrap(wrapped) != cause {
		t.Fatalf("expected direct unwrap to return original cause")
	}
}

func TestWrapAPIConnectionErrorRetainsStableMessageWithoutPubkey(t *testing.T) {
	cause := errors.New("connection refused")
	wrapped := wrapAPIConnectionError(cause)
	if wrapped == nil {
		t.Fatal("expected wrapped error")
	}
	if wrapped.Error() != apiConnectionErrorMessage {
		t.Fatalf("expected stable user-facing message %q, got %q", apiConnectionErrorMessage, wrapped.Error())
	}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("expected wrapped error to preserve cause chain")
	}
}
