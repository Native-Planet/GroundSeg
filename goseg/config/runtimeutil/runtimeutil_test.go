package runtimeutil

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestNetCheckReportsReachableLocalListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate local listener: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	if !NetCheck(listener.Addr().String()) {
		t.Fatal("expected NetCheck to report reachable local listener")
	}
}

func TestRandStringWithErrorHandlesLengthAndEntropy(t *testing.T) {
	empty, err := RandStringWithError(0)
	if err != nil {
		t.Fatalf("expected zero-length random string call to succeed: %v", err)
	}
	if empty != "" {
		t.Fatalf("expected zero-length random string to be empty, got %q", empty)
	}

	secret, err := RandStringWithError(16)
	if err != nil {
		t.Fatalf("expected random string generation to succeed: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty encoded random string")
	}
}

func TestRandStringReturnsBestEffortValue(t *testing.T) {
	value := RandString(16)
	if len(value) == 0 {
		t.Fatal("expected RandString to return a non-empty value")
	}
}

func TestSHA256ReturnsKnownDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash.txt")
	if err := os.WriteFile(path, []byte("groundseg"), 0o644); err != nil {
		t.Fatalf("failed to write hash fixture file: %v", err)
	}

	hash, err := SHA256(path)
	if err != nil {
		t.Fatalf("expected SHA256 to succeed: %v", err)
	}
	const expected = "62bdeec0e24a451b75e5f518384e1ec2856cb5dc2366913d65bb9313899bc2e2"
	if hash != expected {
		t.Fatalf("unexpected digest: want %s got %s", expected, hash)
	}
}
