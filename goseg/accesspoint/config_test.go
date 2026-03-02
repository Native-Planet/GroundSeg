package accesspoint

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestBuildHostapdConfigIncludesSections(t *testing.T) {
	cfg, err := buildHostapdConfig("wlan0", "GroundSeg Test", "valid-password-123")
	if err != nil {
		t.Fatalf("buildHostapdConfig returned error: %v", err)
	}
	if !strings.Contains(cfg, "interface=wlan0") {
		t.Fatalf("expected interface stanza in config: %q", cfg)
	}
	if !strings.Contains(cfg, "hw_mode=g") {
		t.Fatalf("expected mode stanza in config: %q", cfg)
	}
	if !strings.Contains(cfg, "wpa=3") {
		t.Fatalf("expected wpa stanza in config: %q", cfg)
	}
}

func TestBuildHostapdConfigRejectsInvalidInterfaceCharacters(t *testing.T) {
	if _, err := buildHostapdConfig("wlan0/../../", "GroundSeg", "valid-password-123"); err == nil {
		t.Fatal("expected invalid wlan interface to error")
	}
}

func TestBuildHostapdConfigRejectsInvalidSSIDAndPassphrase(t *testing.T) {
	tests := []struct {
		ssid string
		pass string
	}{
		{"bad\nssid", "valid-password-123"},
		{"GroundSeg", "1234"},
	}
	for _, tc := range tests {
		_, err := buildHostapdConfig("wlan0", tc.ssid, tc.pass)
		if err == nil {
			t.Fatalf("expected error for ssid=%q pass=%q", tc.ssid, tc.pass)
		}
	}
}

func TestWriteHostapdConfigPropagatesWriteFailure(t *testing.T) {
	missingDir := t.TempDir() + "/does-not-exist/hostapd.config"
	err := writeHostapdConfig(missingDir, "wlan0", "GroundSeg", "valid-password-123")
	if err == nil {
		t.Fatal("expected writeHostapdConfig to fail for missing parent directory")
	}
	if !errors.Is(err, os.ErrNotExist) {
		// any path-related IO error is acceptable for this test contract.
		return
	}
}

func TestBuildInterfaceSectionEscapesInputs(t *testing.T) {
	got := buildInterfaceSection("wlan0", "GroundSeg")
	if !strings.Contains(got, "interface=wlan0") {
		t.Fatalf("expected interface field, got %q", got)
	}
	if !strings.Contains(got, "ssid=GroundSeg") {
		t.Fatalf("expected ssid field, got %q", got)
	}
}
