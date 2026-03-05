package accesspoint

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"groundseg/internal/testseams"
)

func resetRouterGlobalsForTest(t *testing.T) {
	testseams.WithRestore(t, &wlan, wlan)
	testseams.WithRestore(t, &inet, inet)
	testseams.WithRestore(t, &ip, ip)
	testseams.WithRestore(t, &netmask, netmask)
	testseams.WithRestore(t, &hostapdConfigPath, hostapdConfigPath)
	testseams.WithRestore(t, &executeRouterShellFn, executeRouterShellFn)
	testseams.WithRestore(t, &routerSleepFn, routerSleepFn)
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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

	rt := accessPointRuntime()
	rt.RunProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\nsshd: session\n'")
	}

	running, err := isRunningWithRuntime(rt)
	if err != nil {
		t.Fatalf("expected no error for no process match, got: %v", err)
	}
	if running {
		t.Fatalf("expected accesspoint process probe to return false when no match is found")
	}
}

func TestIsRunningDetectsHostapdAndDnsmasqByName(t *testing.T) {
	resetRouterGlobalsForTest(t)

	rt := accessPointRuntime()
	rt.RunProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\n/usr/sbin/dnsmasq --dhcp-authoritative\n/usr/sbin/hostapd -B /tmp/hostapd.config\n'")
	}

	running, err := isRunningWithRuntime(rt)
	if err != nil {
		t.Fatalf("expected process probe to succeed: %v", err)
	}
	if !running {
		t.Fatalf("expected accesspoint process probe to detect running hostapd/dnsmasq")
	}
}

func TestIsRunningExactMatcherRejectsSubstringMatch(t *testing.T) {
	resetRouterGlobalsForTest(t)

	rt := accessPointRuntime()
	rt.RunProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'COMMAND\nhostapd-wrapper\nmydnsmasq-helper\n'")
	}

	running, err := isRunningWithRuntime(rt)
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
	resetRouterGlobalsForTest(t)

	rt := accessPointRuntime()
	rt.RunProcessProbeFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo failed 1>&2; exit 2")
	}

	running, err := isRunningWithRuntime(rt)
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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

	if got := formatShellCommand("ip"); got != "ip" {
		t.Fatalf("expected bare command string without args, got %q", got)
	}
	if got := formatShellCommand("ip", "link", "set", "wlan0", "up"); got != "ip link set wlan0 up" {
		t.Fatalf("unexpected command formatting output: %q", got)
	}
}

func TestExecuteRouterCommandReturnsShellErrors(t *testing.T) {
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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
	resetRouterGlobalsForTest(t)

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

func TestStopRouterAggregatesTeardownFailures(t *testing.T) {
	resetRouterGlobalsForTest(t)

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
	if !hasCommand(commands, "pkill hostapd") {
		t.Fatalf("expected stopRouter teardown to continue and collect additional errors, commands=%v", commands)
	}
}

func TestStartRouterWithRuntimeForwardsForwardingAndNATErrorPaths(t *testing.T) {
	resetRouterGlobalsForTest(t)

	baseRuntime := AccessPointRuntime{Wlan: "wlan0", Inet: "eth0", IP: "192.168.45.1", Netmask: "255.255.255.0", HostapdConfigPath: "/tmp/hostapd.config"}
	routerSleepFn = func(_ time.Duration) {}

	testCases := []struct {
		name           string
		failCommand    string
		expectedPhrase string
	}{
		{
			name:           "ip_forward_command_fails",
			failCommand:    "sysctl -w net.ipv4.ip_forward=1",
			expectedPhrase: "enable ipv4 forwarding",
		},
		{
			name:           "nat_forward_policy_fails",
			failCommand:    "iptables -P FORWARD ACCEPT",
			expectedPhrase: "set forward policy",
		},
		{
			name:           "nat_delete_chain_fails",
			failCommand:    "iptables --table nat --delete-chain",
			expectedPhrase: "delete nat chain",
		},
		{
			name:           "nat_flush_fails",
			failCommand:    "iptables --table nat -F",
			expectedPhrase: "flush nat table",
		},
		{
			name:           "nat_destroy_fails",
			failCommand:    "iptables --table nat -X",
			expectedPhrase: "delete nat chains",
		},
		{
			name:           "nat_masquerade_fails",
			failCommand:    "iptables -t nat -A POSTROUTING -o wlan0 -j MASQUERADE",
			expectedPhrase: "configure masquerade",
		},
		{
			name:           "nat_forward_established_fails",
			failCommand:    "iptables -A FORWARD -i wlan0 -o wlan0 -j ACCEPT -m state --state RELATED,ESTABLISHED",
			expectedPhrase: "configure forward established allow",
		},
		{
			name:           "nat_forward_accept_fails",
			failCommand:    "iptables -A FORWARD -i wlan0 -o wlan0 -j ACCEPT",
			expectedPhrase: "configure forward accept",
		},
		{
			name:           "output_chain_fails",
			failCommand:    "iptables -A OUTPUT --out-interface wlan0 -j ACCEPT",
			expectedPhrase: "allow output on AP interface",
		},
		{
			name:           "input_chain_fails",
			failCommand:    "iptables -A INPUT --in-interface wlan0 -j ACCEPT",
			expectedPhrase: "allow input on AP interface",
		},
		{
			name:           "dnsmasq_fails",
			failCommand:    "dnsmasq --dhcp-authoritative --interface=wlan0 --dhcp-range=192.168.45.20,192.168.45.100,255.255.255.0,4h",
			expectedPhrase: "start dnsmasq",
		},
		{
			name:           "hostapd_sleep_fails",
			failCommand:    "sleep 2",
			expectedPhrase: "hostapd warmup delay",
		},
		{
			name:           "hostapd_start_fails",
			failCommand:    "hostapd -B /tmp/hostapd.config",
			expectedPhrase: "start hostapd",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var commands []string
			executeRouterShellFn = func(cmd string) (string, error) {
				commands = append(commands, cmd)
				if cmd == tc.failCommand {
					return "", errors.New("execution failed")
				}
				return "", nil
			}

			err := startRouterWithRuntime(baseRuntime)
			if err == nil {
				t.Fatalf("expected startRouterWithRuntime to fail on %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.expectedPhrase) {
				t.Fatalf("expected %s error context, got: %v", tc.expectedPhrase, err)
			}
			if !hasCommand(commands, tc.failCommand) {
				t.Fatalf("expected failure command %s to be executed", tc.failCommand)
			}
		})
	}
}

func TestStartRouterWithRuntimeIpAddrFailure(t *testing.T) {
	resetRouterGlobalsForTest(t)

	routerSleepFn = func(_ time.Duration) {}
	baseRuntime := AccessPointRuntime{Wlan: "wlan0", Inet: "", IP: "192.168.45.1", Netmask: "255.255.255.0", HostapdConfigPath: "/tmp/hostapd.config"}

	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd == "ip addr add 192.168.45.1/24 dev wlan0" {
			return "", errors.New("addr failed")
		}
		return "", nil
	}

	err := startRouterWithRuntime(baseRuntime)
	if err == nil {
		t.Fatal("expected ip addr add error")
	}
	if !strings.Contains(err.Error(), "assign AP IP address") {
		t.Fatalf("expected assign AP IP address context, got: %v", err)
	}
}

func TestStopRouterWithRuntimeSkipsWlanRulesWhenNoInterface(t *testing.T) {
	resetRouterGlobalsForTest(t)

	var commands []string
	executeRouterShellFn = func(cmd string) (string, error) {
		commands = append(commands, cmd)
		return "", nil
	}

	err := stopRouterWithRuntime(AccessPointRuntime{Wlan: "", HostapdConfigPath: "/tmp/hostapd.config"})
	if err != nil {
		t.Fatalf("stopRouterWithRuntime should succeed with empty wlan: %v", err)
	}

	if hasCommand(commands, "iptables -D OUTPUT --out-interface wlan0 -j ACCEPT") {
		t.Fatalf("unexpected wlan-specific OUTPUT rule command when wlan is empty")
	}
	if hasCommand(commands, "iptables -D INPUT --in-interface wlan0 -j ACCEPT") {
		t.Fatalf("unexpected wlan-specific INPUT rule command when wlan is empty")
	}
}

func TestStopRouterWithRuntimeErrorPaths(t *testing.T) {
	resetRouterGlobalsForTest(t)

	testCases := []struct {
		name           string
		wlan           string
		failCommand    string
		expectedPhrase string
	}{
		{
			name:           "forward_policy_fails",
			wlan:           "wlan0",
			failCommand:    "iptables -P FORWARD DROP",
			expectedPhrase: "drop forward policy",
		},
		{
			name:           "remove_output_rule_fails",
			wlan:           "wlan0",
			failCommand:    "iptables -D OUTPUT --out-interface wlan0 -j ACCEPT",
			expectedPhrase: "remove output firewall rule",
		},
		{
			name:           "remove_input_rule_fails",
			wlan:           "wlan0",
			failCommand:    "iptables -D INPUT --in-interface wlan0 -j ACCEPT",
			expectedPhrase: "remove input firewall rule",
		},
		{
			name:           "remove_nat_chain_fails",
			wlan:           "wlan0",
			failCommand:    "iptables --table nat --delete-chain",
			expectedPhrase: "remove nat chain",
		},
		{
			name:           "flush_nat_table_fails",
			wlan:           "wlan0",
			failCommand:    "iptables --table nat -F",
			expectedPhrase: "flush nat table",
		},
		{
			name:           "delete_nat_table_fails",
			wlan:           "wlan0",
			failCommand:    "iptables --table nat -X",
			expectedPhrase: "delete nat table",
		},
		{
			name:           "disable_forwarding_fails",
			wlan:           "wlan0",
			failCommand:    "sysctl -w net.ipv4.ip_forward=0",
			expectedPhrase: "disable ipv4 forwarding",
		},
		{
			name:           "pkill_hostapd_fails",
			wlan:           "wlan0",
			failCommand:    "pkill hostapd",
			expectedPhrase: "stop hostapd",
		},
		{
			name:           "killall_dnsmasq_fails",
			wlan:           "wlan0",
			failCommand:    "killall dnsmasq",
			expectedPhrase: "stop dnsmasq",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var commands []string
			executeRouterShellFn = func(cmd string) (string, error) {
				commands = append(commands, cmd)
				if cmd == tc.failCommand {
					return "", errors.New("execution failed")
				}
				return "", nil
			}

			err := stopRouterWithRuntime(AccessPointRuntime{Wlan: tc.wlan})
			if err == nil {
				t.Fatalf("expected stopRouterWithRuntime to fail on %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.expectedPhrase) {
				t.Fatalf("expected %s error context, got: %v", tc.expectedPhrase, err)
			}
			if !hasCommand(commands, tc.failCommand) {
				t.Fatalf("expected failure command %s to be executed", tc.failCommand)
			}
		})
	}
}

func TestPreStartLegacyNmcliReportsErrorOutput(t *testing.T) {
	resetRouterGlobalsForTest(t)

	executeRouterShellFn = func(cmd string) (string, error) {
		switch cmd {
		case "killall wpa_supplicant":
			return "", nil
		case "nmcli radio wifi off":
			return "error: command failed", nil
		case "nmcli nm wifi off":
			return "error: still failing", nil
		case "rfkill unblock wlan":
			return "", nil
		case "sleep 1":
			return "", nil
		default:
			return "", nil
		}
	}

	err := preStart()
	if err == nil {
		t.Fatal("expected legacy nmcli error output to fail")
	}
	if !strings.Contains(err.Error(), "legacy nmcli wifi disable returned error output") {
		t.Fatalf("unexpected preStart error: %v", err)
	}
}

func TestPreStartFailsIfKillallWpaSupplicantFails(t *testing.T) {
	resetRouterGlobalsForTest(t)

	executeRouterShellFn = func(cmd string) (string, error) {
		if cmd == "killall wpa_supplicant" {
			return "", errors.New("pre-stop failed")
		}
		return "", nil
	}

	err := preStart()
	if err == nil {
		t.Fatal("expected killall wpa_supplicant failure to surface")
	}
	if !strings.Contains(err.Error(), "stop wpa_supplicant") {
		t.Fatalf("expected stop wpa_supplicant context, got: %v", err)
	}
}

func TestPreStartFailsWhenLegacyNmcliCommandFails(t *testing.T) {
	resetRouterGlobalsForTest(t)

	executeRouterShellFn = func(cmd string) (string, error) {
		switch cmd {
		case "killall wpa_supplicant":
			return "", nil
		case "nmcli radio wifi off":
			return "error: wifi radio off failure", nil
		case "nmcli nm wifi off":
			return "", errors.New("legacy command failed")
		case "rfkill unblock wlan":
			return "", nil
		case "sleep 1":
			return "", nil
		default:
			return "", nil
		}
	}

	err := preStart()
	if err == nil {
		t.Fatal("expected legacy nmcli command failure to surface")
	}
	if !strings.Contains(err.Error(), "disable wifi radio (nmcli legacy)") {
		t.Fatalf("expected legacy fallback error context, got: %v", err)
	}
}

func TestPreStartFailsIfDelayFails(t *testing.T) {
	resetRouterGlobalsForTest(t)

	executeRouterShellFn = func(cmd string) (string, error) {
		switch cmd {
		case "sleep 1":
			return "", errors.New("sleep failed")
		case "killall wpa_supplicant", "nmcli radio wifi off", "rfkill unblock wlan":
			return "", nil
		default:
			return "", nil
		}
	}

	err := preStart()
	if err == nil {
		t.Fatal("expected delay command failure to surface")
	}
	if !strings.Contains(err.Error(), "delay before startup") {
		t.Fatalf("expected delay before startup context, got: %v", err)
	}
}
