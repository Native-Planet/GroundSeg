package docker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const validWgConf = `[Interface]
PrivateKey = local-private-key
Address = 10.0.0.2/32

[Peer]
PublicKey = remote-public-key
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
`

func TestWireguardConfigPathFromMountpoint(t *testing.T) {
	path, err := wireguardConfigPathFromMountpoint("/docker/volumes/wireguard/_data")
	if err != nil {
		t.Fatalf("unexpected path error: %v", err)
	}
	if path != "/docker/volumes/wireguard/_data/wg0.conf" {
		t.Fatalf("unexpected WireGuard config path %q", path)
	}
}

func TestWireguardConfigPathRejectsUnsafeMountpoints(t *testing.T) {
	for _, mountpoint := range []string{"", ".", "/", "relative/path"} {
		if _, err := wireguardConfigPathFromMountpoint(mountpoint); err == nil {
			t.Fatalf("expected mountpoint %q to be rejected", mountpoint)
		}
	}
}

func TestWriteWgConfRejectsEmptyContentWithoutTruncatingExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wg0.conf")
	if err := os.WriteFile(path, []byte(validWgConf), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := writeWgConfToFile(path, ""); err == nil {
		t.Fatal("expected empty WireGuard config to be rejected")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(contents) != validWgConf {
		t.Fatalf("existing config was modified: %q", string(contents))
	}
}

func TestSyncWireguardConfigPreservesLastKnownGoodOnRetrieveFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wg0.conf")
	existing := []byte(validWgConf)
	if err := os.WriteFile(path, existing, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	changed, err := syncWireguardConfigAtPath(path, existing, nil, "", errors.New("retrieve failed"))
	if err != nil {
		t.Fatalf("preserve valid existing config: %v", err)
	}
	if changed {
		t.Fatal("preserving the existing config must not report a change")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(contents) != validWgConf {
		t.Fatalf("existing config was modified: %q", string(contents))
	}
}

func TestWriteWgConfAtomicallyReplacesValidContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wg0.conf")
	updated := validWgConf + "PersistentKeepalive = 25\n"
	if err := writeWgConfToFile(path, updated); err != nil {
		t.Fatalf("write config: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(contents) != updated {
		t.Fatalf("unexpected config contents: %q", string(contents))
	}
}
