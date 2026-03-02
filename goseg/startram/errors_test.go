package startram

import (
	"errors"
	"fmt"
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
	if msg != APIConnectionErrorMessage {
		t.Fatalf("expected stable redacted message %q, got %q", APIConnectionErrorMessage, msg)
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
	if wrapped.Error() != APIConnectionErrorMessage {
		t.Fatalf("expected stable user-facing message %q, got %q", APIConnectionErrorMessage, wrapped.Error())
	}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("expected wrapped error to preserve cause chain")
	}
}

func TestWrapAPIConnectionErrorSupportsErrorChainObservability(t *testing.T) {
	inner := errors.New("temporary failure")
	cause := fmt.Errorf("lookup host: %w", inner)
	wrapped := wrapAPIConnectionError(cause)
	if wrapped == nil {
		t.Fatal("expected wrapped error")
	}
	if wrapped.Error() != APIConnectionErrorMessage {
		t.Fatalf("expected stable message %q, got %q", APIConnectionErrorMessage, wrapped.Error())
	}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("expected wrapped error to include original cause")
	}
	if !errors.Is(wrapped, inner) {
		t.Fatal("expected wrapped error chain to expose nested cause")
	}
	if unwrapped := errors.Unwrap(wrapped); unwrapped != cause {
		t.Fatalf("expected direct unwrap to return original cause, got %v", unwrapped)
	}
}

func TestAPIConnectionErrorContractHasMetadata(t *testing.T) {
	contract := APIConnectionErrorContract()
	if contract.Name != "APIConnectionError" {
		t.Fatalf("expected contract name, got %q", contract.Name)
	}
	if contract.ContractMetadata.IntroducedIn == "" {
		t.Fatal("expected introduced version in APIConnectionError contract")
	}
	if contract.Message != APIConnectionErrorMessage {
		t.Fatalf("expected contract message to match stable message")
	}
}

func TestAPIConnectionErrorCompatibilityChecks(t *testing.T) {
	if !IsAPIConnectionErrorContractActive("2026.03.02") {
		t.Fatal("expected API connection error contract to be active at current date")
	}
	if _, ok := APIConnectionErrorContractForVersion("2026.01.01"); ok {
		t.Fatal("expected API connection error contract lookup to fail before introduction version")
	}
	if IsAPIConnectionErrorContractDeprecated("2026.03.02") {
		t.Fatal("did not expect deprecation at current version")
	}
}

func TestAPIConnectionErrorContractVersionBoundaries(t *testing.T) {
	if _, ok := APIConnectionErrorContractForVersion(""); ok {
		t.Fatal("expected empty version to be inactive")
	}
	if _, ok := APIConnectionErrorContractForVersion("bad-version"); ok {
		t.Fatal("expected malformed version to be inactive")
	}
	if _, ok := APIConnectionErrorContractForVersion("2025.12.31"); ok {
		t.Fatal("expected pre-introduction version to be inactive")
	}

	contract, ok := APIConnectionErrorContractForVersion("2026.03.02")
	if !ok {
		t.Fatal("expected current version to be active")
	}
	if contract.Name != "APIConnectionError" {
		t.Fatalf("unexpected contract name: %q", contract.Name)
	}
}
