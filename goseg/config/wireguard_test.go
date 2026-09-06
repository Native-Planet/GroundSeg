package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

const validWireguardTemplate = `[Interface]
PrivateKey = privkey
Address = 10.0.0.2/32

[Peer]
PublicKey = remote-public-key
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
`

func TestDecodeStartramWireguardConfigRejectsEmptyPayload(t *testing.T) {
	if _, err := DecodeStartramWireguardConfig(""); err == nil {
		t.Fatal("expected empty StarTram config to be rejected")
	}
}

func TestDecodeStartramWireguardConfigRejectsStructurallyInvalidPayload(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("not a WireGuard config"))
	if _, err := DecodeStartramWireguardConfig(encoded); err == nil {
		t.Fatal("expected structurally invalid StarTram config to be rejected")
	}
}

func TestDecodeStartramWireguardConfigAcceptsValidTemplate(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(validWireguardTemplate))
	decoded, err := DecodeStartramWireguardConfig(encoded)
	if err != nil {
		t.Fatalf("decode valid template: %v", err)
	}
	if decoded != validWireguardTemplate {
		t.Fatalf("unexpected decoded template: %q", decoded)
	}
}

func TestValidateWireguardConfigRejectsUnreplacedPrivateKey(t *testing.T) {
	err := ValidateWireguardConfig(validWireguardTemplate, false)
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("expected unreplaced private key error, got %v", err)
	}
}
