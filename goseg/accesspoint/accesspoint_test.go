package accesspoint

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/internal/testseams"
)

func TestResolveAPPasswordUsesEnvOverride(t *testing.T) {
	t.Setenv("GROUNDSEG_AP_PASSWORD", "custom-password-123")

	got := resolveAPPassword()
	if got != "custom-password-123" {
		t.Fatalf("expected env override password, got %q", got)
	}
}

func TestResolveAPPasswordFallbackShape(t *testing.T) {
	t.Setenv("GROUNDSEG_AP_PASSWORD", "")

	got := resolveAPPassword()
	if len(got) != 16 {
		t.Fatalf("expected derived password length 16, got %d (%q)", len(got), got)
	}
	if !strings.HasPrefix(got, "np-") {
		t.Fatalf("expected derived password to start with np-, got %q", got)
	}
}

func TestMakeConfigContainsNetworkAndCredentials(t *testing.T) {
	cfg, err := buildHostapdConfig("wlan0", "GroundSegTest", "strong-passphrase")
	if err != nil {
		t.Fatalf("buildHostapdConfig returned error: %v", err)
	}
	if !strings.Contains(cfg, "interface=wlan0") {
		t.Fatalf("expected interface stanza in config: %q", cfg)
	}
	if !strings.Contains(cfg, "ssid=GroundSegTest") {
		t.Fatalf("expected ssid stanza in config: %q", cfg)
	}
	if !strings.Contains(cfg, "wpa_passphrase=strong-passphrase") {
		t.Fatalf("expected passphrase stanza in config: %q", cfg)
	}
}

func TestMakeConfigRejectsInvalidInputs(t *testing.T) {
	testCases := []struct {
		name     string
		wlan     string
		ssid     string
		password string
	}{
		{
			name:     "invalid interface",
			wlan:     "wlan0/../../etc",
			ssid:     "GroundSeg",
			password: "valid-passphrase",
		},
		{
			name:     "invalid ssid",
			wlan:     "wlan0",
			ssid:     "bad\nssid",
			password: "valid-passphrase",
		},
		{
			name:     "invalid passphrase",
			wlan:     "wlan0",
			ssid:     "GroundSeg",
			password: "short",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildHostapdConfig(tc.wlan, tc.ssid, tc.password); err == nil {
				t.Fatal("expected buildHostapdConfig validation error")
			}
		})
	}
}

func TestWriteHostapdConfigWritesConfigFile(t *testing.T) {
	testseams.WithRestore(t, &wlan, "wlan0")
	testseams.WithRestore(t, &ssid, "GroundSegTest")
	testseams.WithRestore(t, &password, "valid-passphrase")
	testseams.WithRestore(t, &hostapdConfigPath, filepath.Join(t.TempDir(), "hostapd.config"))

	wlan = "wlan0"
	ssid = "GroundSegTest"
	password = "valid-passphrase"

	if err := writeHostapdConfig(hostapdConfigPath, wlan, ssid, password); err != nil {
		t.Fatalf("writeHostapdConfig returned error: %v", err)
	}
	data, err := os.ReadFile(hostapdConfigPath)
	if err != nil {
		t.Fatalf("read generated hostapd config: %v", err)
	}
	cfg := string(data)
	if !strings.Contains(cfg, "interface=wlan0") || !strings.Contains(cfg, "ssid=GroundSegTest") || !strings.Contains(cfg, "wpa_passphrase=valid-passphrase") {
		t.Fatalf("unexpected hostapd config contents: %q", cfg)
	}
}

func resetParameterGlobalsForTest(t *testing.T) {
	testseams.WithRestore(t, &wlan, wlan)
	testseams.WithRestore(t, &inet, inet)
	testseams.WithRestore(t, &ip, ip)
	testseams.WithRestore(t, &ssid, ssid)
	testseams.WithRestore(t, &password, password)
}

func TestValidateIP(t *testing.T) {
	if !validateIP("192.168.1.1") {
		t.Fatal("expected valid IPv4 address")
	}
	if validateIP("bad-ip") {
		t.Fatal("expected invalid IP to fail validation")
	}
	if validateIP("2001:db8::1") {
		t.Fatal("expected IPv6 to fail IPv4-only validation")
	}
}

func TestHasInterface(t *testing.T) {
	if !hasInterface([]string{"wlan0", "eth0"}, "wlan0") {
		t.Fatal("expected interface to be found")
	}
	if hasInterface([]string{"wlan0", "eth0"}, "lan0") {
		t.Fatal("expected exact match check, not substring match")
	}
}

func TestCheckParametersValidationPaths(t *testing.T) {
	resetParameterGlobalsForTest(t)

	makeValidRuntime := func() AccessPointRuntime {
		rt := accessPointRuntime()
		rt.NetInterfacesFn = func() ([]net.Interface, error) {
			return []net.Interface{
				{Name: "wlan0"},
				{Name: "eth0"},
			}, nil
		}
		rt.Wlan = "wlan0"
		rt.Inet = "eth0"
		rt.IP = "192.168.45.1"
		rt.SSID = "GroundSeg"
		rt.Password = "supersecret"
		return rt
	}

	t.Run("valid runtime", func(t *testing.T) {
		if err := checkParametersWithContext(makeValidRuntime()); err != nil {
			t.Fatalf("expected valid parameters, got error: %v", err)
		}
	})

	testCases := []struct {
		name   string
		mutate func(*AccessPointRuntime)
	}{
		{
			name: "missing wlan",
			mutate: func(runtime *AccessPointRuntime) {
				runtime.Wlan = "missing"
			},
		},
		{
			name: "missing inet",
			mutate: func(runtime *AccessPointRuntime) {
				runtime.Inet = "missing"
			},
		},
		{
			name: "invalid ip",
			mutate: func(runtime *AccessPointRuntime) {
				runtime.IP = "not-an-ip"
			},
		},
		{
			name: "empty ssid",
			mutate: func(runtime *AccessPointRuntime) {
				runtime.SSID = ""
			},
		},
		{
			name: "empty password",
			mutate: func(runtime *AccessPointRuntime) {
				runtime.Password = ""
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rt := makeValidRuntime()
			tc.mutate(&rt)
			if err := checkParametersWithContext(rt); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}
