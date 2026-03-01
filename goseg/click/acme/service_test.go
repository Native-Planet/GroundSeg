package acme

import (
	"errors"
	"strings"
	"testing"
)

func resetAcmeSeams() {
	executeCommandForAcme = func(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
		return "", nil
	}
}

func TestFixCreateHoonError(t *testing.T) {
	t.Cleanup(resetAcmeSeams)
	executeCommandForAcme = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("write failed")
	}

	err := Fix("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFixExecError(t *testing.T) {
	t.Cleanup(resetAcmeSeams)
	executeCommandForAcme = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("exec failed")
	}

	err := Fix("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFixSuccess(t *testing.T) {
	t.Cleanup(resetAcmeSeams)
	executeCommandForAcme = func(_, _, _, _, _, _ string) (string, error) {
		return "[0 %avow 0 %noun %success]", nil
	}

	if err := Fix("~zod"); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}
