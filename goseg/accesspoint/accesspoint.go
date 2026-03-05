package accesspoint

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type AccessPointRuntime struct {
	Wlan                 string
	Inet                 string
	IP                   string
	Netmask              string
	SSID                 string
	Password             string
	HostapdConfigPath    string
	RootDir              string
	ForceRestart         bool
	CheckDependenciesFn  func() error
	CheckParametersFn    func(AccessPointRuntime) error
	IsRunningFn          func(AccessPointRuntime) (bool, error)
	WriteHostapdConfigFn func(string, string, string, string) error
	EnsureRootDirFn      func(string) error
	StartRouterFn        func(AccessPointRuntime) error
	StopRouterFn         func(AccessPointRuntime) error
	RunProcessProbeFn    func(name string, arg ...string) *exec.Cmd
	NetInterfacesFn      func() ([]net.Interface, error)
}

type resolvedAccessPointRuntime struct {
	runtime AccessPointRuntime
}

var (
	wlan              = "wlan0"
	inet              = ""
	ip                = "192.168.45.1"
	netmask           = "255.255.255.0"
	ssid              = "NativePlanetConnect"
	password          = ""
	rootDir           = "/etc/accesspoint/"
	hostapdConfigPath = filepath.Join(rootDir, "hostapd.config")
)

func defaultAccessPointPassword() string {
	if password != "" {
		return password
	}
	return resolveAPPasswordForContext(wlan, ip)
}

func defaultAccessPointContext() AccessPointRuntime {
	return AccessPointRuntime{
		Wlan:              wlan,
		Inet:              inet,
		IP:                ip,
		Netmask:           netmask,
		SSID:              ssid,
		Password:          defaultAccessPointPassword(),
		RootDir:           rootDir,
		HostapdConfigPath: hostapdConfigPath,
	}
}

func accessPointRuntime() AccessPointRuntime {
	return AccessPointRuntime{
		Wlan:                wlan,
		Inet:                inet,
		IP:                  ip,
		Netmask:             netmask,
		SSID:                ssid,
		Password:            defaultAccessPointPassword(),
		RootDir:             rootDir,
		HostapdConfigPath:   hostapdConfigPath,
		CheckDependenciesFn: checkDependencies,
		CheckParametersFn: func(rt AccessPointRuntime) error {
			return checkParametersWithContext(rt)
		},
		IsRunningFn: func(rt AccessPointRuntime) (bool, error) {
			return isRunningWithRuntime(rt)
		},
		WriteHostapdConfigFn: writeHostapdConfig,
		EnsureRootDirFn:      ensureRootDir,
		StartRouterFn:        startRouterWithRuntime,
		StopRouterFn:         stopRouterWithRuntime,
		RunProcessProbeFn:    exec.Command,
		NetInterfacesFn:      net.Interfaces,
	}
}

func normalizeAccessPointRuntime(rt AccessPointRuntime) AccessPointRuntime {
	if rt.Wlan == "" {
		rt.Wlan = wlan
	}
	if rt.Inet == "" {
		rt.Inet = inet
	}
	if rt.IP == "" {
		rt.IP = ip
	}
	if rt.Netmask == "" {
		rt.Netmask = netmask
	}
	if rt.SSID == "" {
		rt.SSID = ssid
	}
	if rt.Password == "" {
		rt.Password = resolveAPPasswordForContext(rt.Wlan, rt.IP)
	}
	if rt.RootDir == "" {
		rt.RootDir = rootDir
	}
	if rt.HostapdConfigPath == "" {
		rt.HostapdConfigPath = filepath.Join(rt.RootDir, "hostapd.config")
	}
	if rt.CheckDependenciesFn == nil {
		rt.CheckDependenciesFn = checkDependencies
	}
	if rt.CheckParametersFn == nil {
		rt.CheckParametersFn = checkParametersWithContext
	}
	if rt.IsRunningFn == nil {
		rt.IsRunningFn = isRunningWithRuntime
	}
	if rt.WriteHostapdConfigFn == nil {
		rt.WriteHostapdConfigFn = writeHostapdConfig
	}
	if rt.EnsureRootDirFn == nil {
		rt.EnsureRootDirFn = ensureRootDir
	}
	if rt.StartRouterFn == nil {
		rt.StartRouterFn = startRouterWithRuntime
	}
	if rt.StopRouterFn == nil {
		rt.StopRouterFn = stopRouterWithRuntime
	}
	if rt.RunProcessProbeFn == nil {
		rt.RunProcessProbeFn = exec.Command
	}
	if rt.NetInterfacesFn == nil {
		rt.NetInterfacesFn = net.Interfaces
	}
	return rt
}

func resolveAccessPointRuntime(rt AccessPointRuntime) resolvedAccessPointRuntime {
	normalized := normalizeAccessPointRuntime(rt)
	return resolvedAccessPointRuntime{
		runtime: normalized,
	}
}

func StartWithRuntime(rt AccessPointRuntime) error {
	return accessPointLifecycleCoordinator{}.StartResolved(resolveAccessPointRuntime(rt))
}

func StopWithRuntime(rt AccessPointRuntime) error {
	return accessPointLifecycleCoordinator{}.StopResolved(resolveAccessPointRuntime(rt))
}

func resolveAPPassword() string {
	return resolveAPPasswordForContext(wlan, ip)
}

func resolveAPPasswordForContext(wlanName, ipAddress string) string {
	envPassword := strings.TrimSpace(os.Getenv("GROUNDSEG_AP_PASSWORD"))
	if envPassword != "" {
		return envPassword
	}

	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "groundseg"
	}

	seed := fmt.Sprintf("%s|%s|%s", hostname, wlanName, ipAddress)
	hash := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("np-%x", hash)[:16]
}

func Start(dev string) error {
	runtime := accessPointRuntime()
	runtime.Wlan = dev
	return StartWithRuntime(runtime)
}

func Stop(dev string) error {
	runtime := accessPointRuntime()
	runtime.Wlan = dev
	return StopWithRuntime(runtime)
}

func ensureRootDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModePerm)
	}
	return nil
}

func isRunning() (bool, error) {
	return isRunningWithRuntime(accessPointRuntime())
}

type processProbeMatchKind string

const (
	processMatchKindExact processProbeMatchKind = "exact"
	processMatchKindRegex processProbeMatchKind = "regex"
)

type processProbeMatcher struct {
	kind            processProbeMatchKind
	normalizedTerms []string
	regexes         []*regexp.Regexp
}

func newProcessProbeMatcher(kind processProbeMatchKind, terms ...string) (processProbeMatcher, error) {
	if len(terms) == 0 {
		return processProbeMatcher{}, fmt.Errorf("process matcher: missing terms")
	}

	matcher := processProbeMatcher{kind: kind}
	for _, term := range terms {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" {
			return processProbeMatcher{}, fmt.Errorf("process matcher: empty term")
		}
		matcher.normalizedTerms = append(matcher.normalizedTerms, term)
	}

	if kind == processMatchKindRegex {
		for _, term := range matcher.normalizedTerms {
			re, err := regexp.Compile(term)
			if err != nil {
				return processProbeMatcher{}, fmt.Errorf("process matcher: invalid regex term %q: %w", term, err)
			}
			matcher.regexes = append(matcher.regexes, re)
		}
		return matcher, nil
	}

	if kind != processMatchKindExact {
		return processProbeMatcher{}, fmt.Errorf("process matcher: unsupported kind %q", kind)
	}

	return matcher, nil
}

func (m processProbeMatcher) matchesCommandLine(commandLine string) bool {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return false
	}
	if m.kind == processMatchKindExact {
		processName := m.extractProcessName(commandLine)
		for _, term := range m.normalizedTerms {
			if processName == term {
				return true
			}
		}
		return false
	}
	for _, re := range m.regexes {
		if re.MatchString(commandLine) {
			return true
		}
	}
	return false
}

func (m processProbeMatcher) extractProcessName(commandLine string) string {
	commandLine = strings.TrimSpace(commandLine)
	firstField := strings.SplitN(commandLine, " ", 2)[0]
	return strings.ToLower(filepath.Base(firstField))
}

func accessPointProcessMatcher() (processProbeMatcher, error) {
	return newProcessProbeMatcher(processMatchKindExact, "hostapd", "dnsmasq")
}

func isRunningWithMatcher(matcher processProbeMatcher, rt AccessPointRuntime) (bool, error) {
	rt = normalizeAccessPointRuntime(rt)
	if rt.RunProcessProbeFn == nil {
		return false, fmt.Errorf("missing process probe runtime")
	}
	out, err := rt.RunProcessProbeFn("ps", "-eo", "args").Output()
	if err != nil {
		return false, fmt.Errorf("process probe failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "COMMAND" {
			continue
		}
		if matcher.matchesCommandLine(line) {
			return true, nil
		}
	}
	return false, nil
}

func isRunningWithRuntime(rt AccessPointRuntime) (bool, error) {
	matcher, err := accessPointProcessMatcher()
	if err != nil {
		return false, fmt.Errorf("build process matcher: %w", err)
	}
	return isRunningWithMatcher(matcher, rt)
}

func checkDependencies() error {
	if _, err := exec.LookPath("hostapd"); err != nil {
		return fmt.Errorf("failed to locate hostapd: %w", err)
	}
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		return fmt.Errorf("failed to locate dnsmasq: %w", err)
	}
	return nil
}

// ExecuteShell executes a shell command and returns its output
func executeShell(commandString string) (string, error) {
	zap.L().Debug(fmt.Sprintf("%v", commandString))
	// Initialize the command
	cmd := exec.Command("sh", "-c", commandString)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command %q failed with exit status %d: %w: %s", commandString, exitErr.ExitCode(), exitErr, stderrText)
		}
		return "", fmt.Errorf("command %q failed: %w: %s", commandString, err, stderrText)
	}

	// Decode the result
	zap.L().Debug(stdout.String())
	return stdout.String(), nil
}

func checkParametersWithRuntime(rt AccessPointRuntime) error {
	return checkParametersWithContext(rt)
}
