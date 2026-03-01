package accesspoint

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	cfg, err := makeConfig("wlan0", "GroundSegTest", "strong-passphrase")
	if err != nil {
		t.Fatalf("makeConfig returned error: %v", err)
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
			if _, err := makeConfig(tc.wlan, tc.ssid, tc.password); err == nil {
				t.Fatal("expected makeConfig validation error")
			}
		})
	}
}

func TestWriteHostapdConfigWritesConfigFile(t *testing.T) {
	originalWlan := wlan
	originalSSID := ssid
	originalPassword := password
	originalPath := hostapdConfigPath
	t.Cleanup(func() {
		wlan = originalWlan
		ssid = originalSSID
		password = originalPassword
		hostapdConfigPath = originalPath
	})

	wlan = "wlan0"
	ssid = "GroundSegTest"
	password = "valid-passphrase"
	hostapdConfigPath = filepath.Join(t.TempDir(), "hostapd.config")

	if err := writeHostapdConfig(); err != nil {
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

func resetParameterGlobalsForTest() func() {
	originalWlan := wlan
	originalInet := inet
	originalIP := ip
	originalSSID := ssid
	originalPassword := password
	originalInterfacesFn := netInterfacesFn
	return func() {
		wlan = originalWlan
		inet = originalInet
		ip = originalIP
		ssid = originalSSID
		password = originalPassword
		netInterfacesFn = originalInterfacesFn
	}
}

func TestValidateIP(t *testing.T) {
	if !validateIP("192.168.1.1") {
		t.Fatal("expected valid IPv4 address")
	}
	if validateIP("bad-ip") {
		t.Fatal("expected invalid IP to fail validation")
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
	restore := resetParameterGlobalsForTest()
	t.Cleanup(restore)

	netInterfacesFn = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "wlan0"},
			{Name: "eth0"},
		}, nil
	}

	wlan = "wlan0"
	inet = "eth0"
	ip = "192.168.45.1"
	ssid = "GroundSeg"
	password = "supersecret"

	if err := checkParameters(); err != nil {
		t.Fatalf("expected valid parameters, got error: %v", err)
	}

	wlan = "missing"
	if err := checkParameters(); err == nil {
		t.Fatal("expected missing wlan validation error")
	}
	wlan = "wlan0"

	inet = "missing"
	if err := checkParameters(); err == nil {
		t.Fatal("expected missing inet validation error")
	}
	inet = "eth0"

	ip = "not-an-ip"
	if err := checkParameters(); err == nil {
		t.Fatal("expected invalid ip validation error")
	}
	ip = "192.168.45.1"

	ssid = ""
	if err := checkParameters(); err == nil {
		t.Fatal("expected empty ssid validation error")
	}
	ssid = "GroundSeg"

	password = ""
	if err := checkParameters(); err == nil {
		t.Fatal("expected empty password validation error")
	}
}

func resetRouterGlobalsForTest() func() {
	originalWlan := wlan
	originalInet := inet
	originalIP := ip
	originalNetmask := netmask
	originalHostapdPath := hostapdConfigPath
	originalExec := executeRouterShellFn
	originalSleep := routerSleepFn
	return func() {
		wlan = originalWlan
		inet = originalInet
		ip = originalIP
		netmask = originalNetmask
		hostapdConfigPath = originalHostapdPath
		executeRouterShellFn = originalExec
		routerSleepFn = originalSleep
	}
}

func hasCommand(commands []string, want string) bool {
	for _, cmd := range commands {
		if cmd == want {
			return true
		}
	}
	return false
}

func TestPreStartUsesLegacyNmcliFallbackOnError(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	var commands []string
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		if cmd == "nmcli radio wifi off" {
			return "Error: unsupported", nil
		}
		return "", nil
	}

	preStart()

	if !hasCommand(commands, "nmcli nm wifi off") {
		t.Fatalf("expected legacy nmcli fallback command, got %v", commands)
	}
}

func TestStartRouterBuildsExpectedCommandFlow(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	inet = "eth0"
	ip = "192.168.45.1"
	netmask = "255.255.255.0"
	hostapdConfigPath = "/tmp/hostapd.config"
	routerSleepFn = func(_ time.Duration) {}

	var commands []string
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		return "", nil
	}

	if !startRouter() {
		t.Fatal("startRouter should return true")
	}
	expected := []string{
		"ip link set wlan0 up",
		"ip addr add 192.168.45.1/24 dev wlan0",
		"sysctl -w net.ipv4.ip_forward=1",
		"iptables -P FORWARD ACCEPT",
		"dnsmasq --dhcp-authoritative --interface=wlan0 --dhcp-range=192.168.45.20,192.168.45.100,255.255.255.0,4h",
		"hostapd -B /tmp/hostapd.config",
	}
	for _, want := range expected {
		if !hasCommand(commands, want) {
			t.Fatalf("missing expected command %q in %v", want, commands)
		}
	}
}

func TestStopRouterBuildsExpectedTeardownCommands(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	var commands []string
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		return "", nil
	}

	if !stopRouter() {
		t.Fatal("stopRouter should return true")
	}
	expected := []string{
		"ip link set wlan0 down",
		"pkill hostapd",
		"killall dnsmasq",
		"iptables -P FORWARD DROP",
		"iptables -D OUTPUT --out-interface wlan0 -j ACCEPT",
		"iptables -D INPUT --in-interface wlan0 -j ACCEPT",
		"sysctl -w net.ipv4.ip_forward=0",
	}
	for _, want := range expected {
		if !hasCommand(commands, want) {
			t.Fatalf("missing expected teardown command %q in %v", want, commands)
		}
	}
}
