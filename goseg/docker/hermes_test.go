package docker

import (
	"testing"

	"groundseg/structs"
)

func TestHermesShipTargetUsesRemoteURLForWireguardShips(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network: "wireguard",
		WgURL:   "sampel-palnet.nativeplanet.live",
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "https://sampel-palnet.nativeplanet.live" {
		t.Fatalf("expected clickable ship URL to use remote HTTPS URL, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected remote target, got error: %v", err)
	}
	if target.URL != "https://sampel-palnet.nativeplanet.live" {
		t.Fatalf("expected remote HTTPS URL, got %q", target.URL)
	}
	if len(target.ExtraHosts) != 0 {
		t.Fatalf("expected no extra host mapping for remote URL, got %#v", target.ExtraHosts)
	}
}

func TestHermesShipTargetUsesCustomRemoteURLWhenSelected(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network:        "wireguard",
		WgURL:          "sampel-palnet.nativeplanet.live",
		CustomUrbitWeb: "chat.example.com",
		ShowUrbitWeb:   "custom",
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "https://chat.example.com" {
		t.Fatalf("expected clickable ship URL to use custom alias, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected custom remote target, got error: %v", err)
	}
	if target.URL != "https://chat.example.com" {
		t.Fatalf("expected custom HTTPS URL, got %q", target.URL)
	}
}

func TestHermesShipTargetKeepsSchemeOnCustomRemoteURL(t *testing.T) {
	target, err := hermesShipTargetForContainer(structs.UrbitDocker{
		Network:        "wireguard",
		WgURL:          "sampel-palnet.nativeplanet.live",
		CustomUrbitWeb: "https://chat.example.com",
		ShowUrbitWeb:   "custom",
	})
	if err != nil {
		t.Fatalf("expected custom remote target, got error: %v", err)
	}
	if target.URL != "https://chat.example.com" {
		t.Fatalf("expected custom URL scheme to be preserved, got %q", target.URL)
	}
}

func TestHermesShipTargetUsesHostGatewayForLocalShips(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network:  "bridge",
		HTTPPort: 8080,
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "http://nativeplanet.local:8080" {
		t.Fatalf("expected clickable ship URL to use local host URL, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected local target, got error: %v", err)
	}
	if target.URL != "http://host.docker.internal:8080" {
		t.Fatalf("expected host-gateway URL, got %q", target.URL)
	}
	if len(target.ExtraHosts) != 1 || target.ExtraHosts[0] != "host.docker.internal:host-gateway" {
		t.Fatalf("expected host-gateway extra host, got %#v", target.ExtraHosts)
	}
}
