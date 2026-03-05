package accesspoint

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func restoreRouterCommandGlobals() func() {
	originalRouterShell := executeRouterShellFn
	originalRouterSleep := routerSleepFn
	return func() {
		executeRouterShellFn = originalRouterShell
		routerSleepFn = originalRouterSleep
	}
}

func TestFormatShellCommandForRouter(t *testing.T) {
	if got := formatShellCommand("ip"); got != "ip" {
		t.Fatalf("expected name only, got %q", got)
	}
	if got := formatShellCommand("ip", "link", "set", "wlan0", "up"); got != "ip link set wlan0 up" {
		t.Fatalf("unexpected shell command: %q", got)
	}
}

func TestIPv4NetworkPrefix(t *testing.T) {
	ipv4Prefix, err := ipv4NetworkPrefix("192.168.45.1")
	if err != nil {
		t.Fatalf("expected ipv4 prefix, got: %v", err)
	}
	if ipv4Prefix != "192.168.45" {
		t.Fatalf("expected prefix 192.168.45, got %q", ipv4Prefix)
	}

	if _, err := ipv4NetworkPrefix("2001:db8::1"); err == nil {
		t.Fatal("expected non-IPv4 address to fail network prefix extraction")
	}
}

func TestStartRouterWithRuntimeRejectsIPv6Address(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	routerSleepFn = func(_ time.Duration) {}
	var commands []string
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		return "", nil
	}

	err := startRouterWithRuntime(AccessPointRuntime{
		Wlan:              "wlan0",
		IP:                "2001:db8::1",
		Netmask:           "255.255.255.0",
		HostapdConfigPath: "/tmp/hostapd.config",
	})
	if err == nil {
		t.Fatal("expected startRouterWithRuntime to reject IPv6 IP")
	}
	if !strings.Contains(err.Error(), "start router") {
		t.Fatalf("expected wrapped start router error, got: %v", err)
	}
	if len(commands) != 0 {
		t.Fatalf("expected no router commands when IP validation fails, got %v", commands)
	}
}

func TestExecuteRouterShellCommand(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	output := "ok\n"
	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd != "echo hello" {
			t.Fatalf("unexpected command: %q", cmd)
		}
		return output, nil
	}

	got, err := executeRouterShellCommand("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != output {
		t.Fatalf("expected %q, got %q", output, got)
	}
}

func TestExecuteRouterCommand(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	calls := 0
	executeRouterShellFn = func(cmd string) (string, error) {
		calls++
		return "ok", nil
	}

	if err := executeRouterCommand("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		t.Fatalf("executeRouterCommand returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one shell invocation, got %d", calls)
	}
}

func TestExecuteRouterCommandPropagatesError(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	execErr := errors.New("command failed")
	executeRouterShellFn = func(_ string) (string, error) {
		return "", execErr
	}
	if err := executeRouterCommand("iptables", "-A", "INPUT"); err == nil {
		t.Fatal("expected executeRouterCommand to propagate failure")
	}
}

func TestPreStartSuccess(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	var seen []string
	executeRouterShellFn = func(cmd string) (string, error) {
		seen = append(seen, cmd)
		return "", nil
	}
	routerSleepFn = func(_ time.Duration) {}

	if err := preStart(); err != nil {
		t.Fatalf("preStart returned error: %v", err)
	}
	if len(seen) != 4 {
		t.Fatalf("expected 4 pre-start commands, got %d: %+v", len(seen), seen)
	}
}

func TestPreStartFallsBackWhenNmcliReturnsError(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	var seen []string
	executeRouterShellFn = func(cmd string) (string, error) {
		seen = append(seen, cmd)
		if len(seen) == 2 {
			return "some error from nmcli", nil
		}
		return "", nil
	}

	if err := preStart(); err != nil {
		t.Fatalf("preStart returned error: %v", err)
	}
	if len(seen) != 5 {
		t.Fatalf("expected fallback command flow of 5 commands, got %d: %+v", len(seen), seen)
	}
}

func TestPreStartLegacyFailsOnErrorOutput(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	call := 0
	executeRouterShellFn = func(cmd string) (string, error) {
		call++
		if call == 1 {
			return "", nil
		}
		if call == 2 {
			return "error", nil
		}
		if call == 3 {
			return "legacy error", nil
		}
		if call == 4 || call == 5 {
			return "", nil
		}
		return "", nil
	}

	if err := preStart(); err == nil {
		t.Fatal("expected preStart to fail on legacy error output")
	}
}

func TestPreStartPropagatesCommandError(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	executeRouterShellFn = func(cmd string) (string, error) {
		return "", errors.New("shell down")
	}
	if err := preStart(); err == nil {
		t.Fatal("expected preStart command failure")
	}
}

func TestStartRouterWithRuntimeNoNat(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	rt := AccessPointRuntime{
		Wlan:    "wlan0",
		IP:      "192.168.45.1",
		Inet:    "",
		Netmask: "255.255.255.0",
	}

	var seen []string
	executeRouterShellFn = func(cmd string) (string, error) {
		seen = append(seen, cmd)
		return "", nil
	}
	routerSleepFn = func(_ time.Duration) {}

	if err := startRouterWithRuntime(rt); err != nil {
		t.Fatalf("startRouterWithRuntime returned error: %v", err)
	}

	if !containsCommandPrefix(seen, "ip link set wlan0 up") {
		t.Fatal("expected ip link up command")
	}
	if !containsCommandPrefix(seen, "iptables -A OUTPUT --out-interface wlan0 -j ACCEPT") {
		t.Fatal("expected iptables OUTPUT command")
	}
	if !containsCommandPrefix(seen, "iptables -A INPUT --in-interface wlan0 -j ACCEPT") {
		t.Fatal("expected iptables INPUT command")
	}
	if !containsCommandPrefix(seen, "dnsmasq --dhcp-authoritative --interface=wlan0 --dhcp-range=192.168.45.20,192.168.45.100,255.255.255.0,4h") {
		t.Fatal("expected dnsmasq command")
	}
	if !containsCommandPrefix(seen, "hostapd -B ") {
		t.Fatal("expected hostapd command")
	}
}

func TestStartRouterWithRuntimeNatBranch(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	rt := AccessPointRuntime{
		Wlan:    "wlan0",
		IP:      "192.168.45.1",
		Inet:    "eth0",
		Netmask: "255.255.255.0",
	}

	var seen []string
	executeRouterShellFn = func(cmd string) (string, error) {
		seen = append(seen, cmd)
		return "", nil
	}
	routerSleepFn = func(_ time.Duration) {}

	if err := startRouterWithRuntime(rt); err != nil {
		t.Fatalf("startRouterWithRuntime returned error: %v", err)
	}

	if !containsCommandPrefix(seen, "iptables -P FORWARD ACCEPT") {
		t.Fatal("expected NAT forward policy command")
	}
	if !containsCommandPrefix(seen, "iptables --table nat --delete-chain") {
		t.Fatal("expected nat delete-chain command")
	}
	if !containsCommandPrefix(seen, "iptables -t nat -A POSTROUTING -o wlan0 -j MASQUERADE") {
		t.Fatal("expected nat masquerade command")
	}
}

func TestStartRouterWithRuntimePropagatesExecutionError(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	executeRouterShellFn = func(cmd string) (string, error) {
		if strings.HasPrefix(cmd, "killall wpa_supplicant") {
			return "ok", nil
		}
		return "", errors.New("command failed")
	}
	routerSleepFn = func(_ time.Duration) {}

	if err := startRouterWithRuntime(AccessPointRuntime{
		Wlan:    "wlan0",
		IP:      "192.168.45.1",
		Netmask: "255.255.255.0",
	}); err == nil {
		t.Fatal("expected startRouterWithRuntime to propagate command failure")
	}
}

func TestStopRouterWithRuntime(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	var seen []string
	executeRouterShellFn = func(cmd string) (string, error) {
		seen = append(seen, cmd)
		return "", nil
	}

	if err := stopRouterWithRuntime(AccessPointRuntime{Wlan: "wlan0"}); err != nil {
		t.Fatalf("stopRouterWithRuntime returned error: %v", err)
	}
	if !containsCommandPrefix(seen, "ip link set wlan0 down") {
		t.Fatal("expected interface down command")
	}
	if !containsCommandPrefix(seen, "pkill hostapd") {
		t.Fatal("expected hostapd stop command")
	}
	if !containsCommandPrefix(seen, "killall dnsmasq") {
		t.Fatal("expected dnsmasq kill command")
	}
	if !containsCommandPrefix(seen, "iptables -D OUTPUT --out-interface wlan0 -j ACCEPT") {
		t.Fatal("expected INPUT rule removal command")
	}
	if !containsCommandPrefix(seen, "iptables --table nat -X") {
		t.Fatal("expected nat table command")
	}
}

func TestStopRouterWithRuntimePropagatesFailure(t *testing.T) {
	restore := restoreRouterCommandGlobals()
	t.Cleanup(restore)

	executeRouterShellFn = func(cmd string) (string, error) {
		if strings.Contains(cmd, "ip link set") {
			return "", nil
		}
		return "", errors.New("failed")
	}
	if err := stopRouterWithRuntime(AccessPointRuntime{Wlan: "wlan0"}); err == nil {
		t.Fatal("expected stopRouterWithRuntime failure")
	}
}

func containsCommandPrefix(commands []string, expected string) bool {
	for _, command := range commands {
		if strings.HasPrefix(command, expected) {
			return true
		}
	}
	return false
}
