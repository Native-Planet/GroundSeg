package accesspoint

import (
	"errors"
	"net"
	"os"
	"os/exec"
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
	cfg, err := buildHostapdConfig("wlan0", "GroundSegTest", "strong-passphrase")
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
			if _, err := buildHostapdConfig(tc.wlan, tc.ssid, tc.password); err == nil {
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
	originalProbe := runProcessProbeFn
	originalSleep := routerSleepFn
	return func() {
		wlan = originalWlan
		inet = originalInet
		ip = originalIP
		netmask = originalNetmask
		hostapdConfigPath = originalHostapdPath
		executeRouterShellFn = originalExec
		runProcessProbeFn = originalProbe
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

	if err := preStart(); err != nil {
		t.Fatalf("preStart should recover via legacy fallback: %v", err)
	}

	if !hasCommand(commands, "nmcli nm wifi off") {
		t.Fatalf("expected legacy nmcli fallback command, got %v", commands)
	}
}

func TestIsRunningReturnsFalseOnNoMatch(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	runProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\nsshd: session\n'")
	}

	running, err := isRunning()
	if err != nil {
		t.Fatalf("expected no error for no process match, got: %v", err)
	}
	if running {
		t.Fatalf("expected accesspoint process probe to return false when no match is found")
	}
}

func TestIsRunningDetectsHostapdAndDnsmasqByName(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	runProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\n/usr/sbin/dnsmasq --dhcp-authoritative\n/usr/sbin/hostapd -B /tmp/hostapd.config\n'")
	}

	running, err := isRunning()
	if err != nil {
		t.Fatalf("expected process probe to succeed: %v", err)
	}
	if !running {
		t.Fatalf("expected accesspoint process probe to detect running hostapd/dnsmasq")
	}
}

func TestIsRunningExactMatcherRejectsSubstringMatch(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	runProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\nhostapd-wrapper\nmydnsmasq-helper\n'")
	}

	running, err := isRunning()
	if err != nil {
		t.Fatalf("expected process probe to succeed: %v", err)
	}
	if running {
		t.Fatalf("expected exact matcher to ignore substring process names")
	}
}

func TestProcessProbeMatcherRegexMatchesCommandLine(t *testing.T) {
	matcher, err := newProcessProbeMatcher(processMatchKindRegex, `(^|/)dnsmasq(\s|$)`, `hosta.*d`)
	if err != nil {
		t.Fatalf("expected matcher constructor to succeed: %v", err)
	}

	if !matcher.matchesCommandLine("/usr/sbin/dnsmasq --dhcp-authoritative") {
		t.Fatalf("expected regex matcher to detect dnsmasq process command line")
	}
	if !matcher.matchesCommandLine("hostapd -B /tmp/hostapd.config") {
		t.Fatalf("expected regex matcher to detect hostapd process command line")
	}
	if matcher.matchesCommandLine("myhostahelper") {
		t.Fatalf("expected regex matcher to ignore non-matching command line")
	}
}

func TestNewProcessProbeMatcherRejectsEmptyTerms(t *testing.T) {
	if _, err := newProcessProbeMatcher(processMatchKindExact); err == nil {
		t.Fatal("expected matcher creation to fail without terms")
	}
	if _, err := newProcessProbeMatcher(processMatchKindRegex, "("); err == nil {
		t.Fatal("expected invalid regex matcher term to fail")
	}
}

func TestIsRunningPropagatesProbeFailure(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	runProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo failed 1>&2; exit 2")
	}

	running, err := isRunning()
	if err == nil {
		t.Fatalf("expected isRunning to propagate probe failure, got nil error")
	}
	if running {
		t.Fatalf("expected running=false on probe failure, got true")
	}
	if !strings.Contains(err.Error(), "process probe failed") {
		t.Fatalf("expected wrapped process probe error, got: %v", err)
	}
}

func TestPreStartPropagatesFailedCommandError(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd == "rfkill unblock wlan" {
			return "", errors.New("rfkill failed")
		}
		return "", nil
	}

	err := preStart()
	if err == nil {
		t.Fatal("expected preStart to fail when rfkill fails")
	}
	if !strings.Contains(err.Error(), "unblock wlan interface") {
		t.Fatalf("expected wrapped unblock error, got %v", err)
	}
}

func TestStartRouterFailsWhenPreStartFails(t *testing.T) {
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
		if cmd == "nmcli radio wifi off" {
			return "", errors.New("wifi command failed")
		}
		return "", nil
	}

	err := startRouter()
	if err == nil {
		t.Fatal("expected startRouter to fail on preStart error")
	}
	if !strings.Contains(err.Error(), "pre-start configuration") {
		t.Fatalf("expected contextual pre-start error, got: %v", err)
	}
	if hasCommand(commands, "ip link set wlan0 up") {
		t.Fatalf("expected startup to stop after preStart failure, got commands: %v", commands)
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

	if err := startRouter(); err != nil {
		t.Fatalf("startRouter should succeed: %v", err)
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

func TestExecuteShellReturnsCommandContextOnError(t *testing.T) {
	output, err := executeShell("echo failed! 1>&2; exit 7")
	if err == nil {
		t.Fatal("expected executeShell to return an error")
	}
	if output != "" {
		t.Fatalf("expected no stdout output, got: %q", output)
	}
	if !strings.Contains(err.Error(), "echo failed! 1>&2; exit 7") {
		t.Fatalf("expected command context in error: %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("expected exit status in error: %v", err)
	}
	if !strings.Contains(err.Error(), "failed!") {
		t.Fatalf("expected stderr in error: %v", err)
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

	if err := stopRouter(); err != nil {
		t.Fatalf("stopRouter should succeed: %v", err)
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

func TestStopRouterReturnsConcreteError(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd == "ip link set wlan0 down" {
			return "", errors.New("down failed")
		}
		return "", nil
	}

	err := stopRouter()
	if err == nil {
		t.Fatal("expected stopRouter to fail on first command error")
	}
	if !strings.Contains(err.Error(), "stop wlan interface") {
		t.Fatalf("expected contextual router error, got: %v", err)
	}
}

func TestFormatShellCommand(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	if got := formatShellCommand("ip"); got != "ip" {
		t.Fatalf("expected bare command string without args, got %q", got)
	}
	if got := formatShellCommand("ip", "link", "set", "wlan0", "up"); got != "ip link set wlan0 up" {
		t.Fatalf("unexpected command formatting output: %q", got)
	}
}

func TestExecuteRouterCommandReturnsShellErrors(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd == "ip link set wlan0 up" {
			return "route down", errors.New("execution failed")
		}
		return "", nil
	}

	err := executeRouterCommand("ip", "link", "set", "wlan0", "up")
	if err == nil {
		t.Fatal("expected executeRouterCommand to return shell error")
	}
	if !strings.Contains(err.Error(), "execution failed") {
		t.Fatalf("expected original error context in wrapped output, got: %v", err)
	}
}

func TestStartRouterWithRuntimeSkipsNatCommandsWithoutInternetBridge(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	inet = ""
	ip = "192.168.45.1"
	netmask = "255.255.255.0"
	routerSleepFn = func(_ time.Duration) {}

	commands := []string{}
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		return "", nil
	}

	rt := AccessPointRuntime{Wlan: "wlan0", Inet: "", IP: "192.168.45.1", Netmask: "255.255.255.0"}
	if err := startRouterWithRuntime(rt); err != nil {
		t.Fatalf("startRouterWithRuntime should skip NAT branch without inet: %v", err)
	}

	for _, forbidden := range []string{
		"iptables -P FORWARD ACCEPT",
		"iptables --table nat --delete-chain",
		"iptables -t nat -F",
		"iptables -t nat -X",
		"iptables -A FORWARD -i wlan0 -o wlan0 -j ACCEPT -m state --state RELATED,ESTABLISHED",
		"iptables -A FORWARD -i wlan0 -o wlan0 -j ACCEPT",
	} {
		if hasCommand(commands, forbidden) {
			t.Fatalf("unexpected NAT command in startup flow: %q in %v", forbidden, commands)
		}
	}

	if !hasCommand(commands, "dnsmasq --dhcp-authoritative --interface=wlan0 --dhcp-range=192.168.45.20,192.168.45.100,255.255.255.0,4h") {
		t.Fatalf("expected dnsmasq command still present when skipping NAT branch")
	}
}

func TestStartRouterHaltsOnFirstIpFailure(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	inet = "eth0"
	ip = "192.168.45.1"
	netmask = "255.255.255.0"
	routerSleepFn = func(_ time.Duration) {}

	commands := []string{}
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		if cmd == "ip link set wlan0 up" {
			return "", errors.New("start failed")
		}
		return "", nil
	}

	err := startRouter()
	if err == nil {
		t.Fatal("expected startRouter to fail on initial ip setup failure")
	}
	if !strings.Contains(err.Error(), "start wlan interface") {
		t.Fatalf("expected contextual setup error, got: %v", err)
	}
	if hasCommand(commands, "ip addr add 192.168.45.1/24 dev wlan0") {
		t.Fatalf("expected startup to stop after first failed ip command: %v", commands)
	}
}

func TestStopRouterHaltsOnFirstTeardownFailure(t *testing.T) {
	restore := resetRouterGlobalsForTest()
	t.Cleanup(restore)

	wlan = "wlan0"
	commands := []string{}
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		if cmd == "ip link set wlan0 down" {
			return "", errors.New("stop failed")
		}
		return "", nil
	}

	err := stopRouter()
	if err == nil {
		t.Fatal("expected stopRouter to fail on initial teardown command")
	}
	if !strings.Contains(err.Error(), "stop wlan interface") {
		t.Fatalf("expected contextual teardown error, got: %v", err)
	}
	if hasCommand(commands, "pkill hostapd") {
		t.Fatalf("expected stopRouter to halt after first teardown failure")
	}
}
