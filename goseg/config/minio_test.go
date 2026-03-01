package config

import (
	"strings"
	"testing"
)

func resetMinIOPasswords() {
	minioPwdMutex.Lock()
	minIOPasswords = make(map[string]string)
	minioPwdMutex.Unlock()
}

func TestMinIOPasswordStore(t *testing.T) {
	t.Cleanup(resetMinIOPasswords)

	if _, err := GetMinIOPassword("alice"); err == nil || !strings.Contains(err.Error(), "alice password does not exist") {
		t.Fatalf("expected missing password error, got %v", err)
	}

	if err := SetMinIOPassword("alice", "secret"); err != nil {
		t.Fatalf("SetMinIOPassword failed: %v", err)
	}
	got, err := GetMinIOPassword("alice")
	if err != nil {
		t.Fatalf("GetMinIOPassword failed: %v", err)
	}
	if got != "secret" {
		t.Fatalf("unexpected password: %s", got)
	}
}
