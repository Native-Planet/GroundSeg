package accesspoint

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
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
}

var (
	wlan                 = "wlan0"
	inet                 = ""
	ip                   = "192.168.45.1"
	netmask              = "255.255.255.0"
	ssid                 = "NativePlanetConnect"
	password             = resolveAPPassword()
	rootDir              = "/etc/accesspoint/"
	hostapdConfigPath    = filepath.Join(rootDir, "hostapd.config")
	runProcessProbeFn    = exec.Command
	checkDependenciesFn  = checkDependencies
	writeHostapdConfigFn = writeHostapdConfig
	checkParametersFn    = func() error {
		return checkParametersWithContext(defaultAccessPointContext())
	}
	isRunningFn = func() (bool, error) {
		return isRunningWithRuntime(defaultAccessPointContext())
	}
)

func defaultAccessPointContext() AccessPointRuntime {
	return AccessPointRuntime{
		Wlan:              wlan,
		Inet:              inet,
		IP:                ip,
		Netmask:           netmask,
		SSID:              ssid,
		Password:          password,
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
		Password:            password,
		RootDir:             rootDir,
		HostapdConfigPath:   hostapdConfigPath,
		CheckDependenciesFn: checkDependenciesFn,
		CheckParametersFn: func(rt AccessPointRuntime) error {
			return checkParametersFn()
		},
		IsRunningFn: func(rt AccessPointRuntime) (bool, error) {
			return isRunningFn()
		},
		WriteHostapdConfigFn: writeHostapdConfigFn,
		EnsureRootDirFn:      ensureRootDir,
		StartRouterFn:        startRouterWithRuntime,
		StopRouterFn:         stopRouterWithRuntime,
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
		rt.CheckDependenciesFn = checkDependenciesFn
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
	return rt
}

func StartWithRuntime(rt AccessPointRuntime) error {
	rt = normalizeAccessPointRuntime(rt)
	return startWithRuntime(rt)
}

func StopWithRuntime(rt AccessPointRuntime) error {
	rt = normalizeAccessPointRuntime(rt)
	return stopWithRuntime(rt)
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

func startWithRuntime(rt AccessPointRuntime) error {
	zap.L().Info(fmt.Sprintf("Starting router on %v", rt.Wlan))
	if rt.EnsureRootDirFn == nil {
		return errors.New("missing root directory init runtime")
	}
	if err := rt.EnsureRootDirFn(rt.RootDir); err != nil {
		return err
	}
	// make sure dependencies are met
	if rt.CheckDependenciesFn == nil {
		return errors.New("missing dependency runtime")
	}
	if err := rt.CheckDependenciesFn(); err != nil {
		return err
	}
	// make sure params are set (maybe not needed)
	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation runtime")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP already running
	if rt.IsRunningFn == nil {
		return errors.New("missing status runtime")
	}
	running, err := rt.IsRunningFn(rt)
	if err != nil {
		return err
	}
	if running {
		if rt.ForceRestart {
			zap.L().Info("Accesspoint already started; force restart requested")
		} else {
			zap.L().Info("Accesspoint already started")
			return nil
		}
	}
	// dump config to file
	if rt.WriteHostapdConfigFn == nil {
		return errors.New("missing hostapd config runtime")
	}
	if err := rt.WriteHostapdConfigFn(rt.HostapdConfigPath, rt.Wlan, rt.SSID, rt.Password); err != nil {
		return err
	}
	// start the router
	if rt.StartRouterFn == nil {
		return errors.New("missing router start runtime")
	}
	if err := rt.StartRouterFn(rt); err != nil {
		return fmt.Errorf("start router: %w", err)
	}
	return nil
}

func Stop(dev string) error {
	runtime := accessPointRuntime()
	runtime.Wlan = dev
	return StopWithRuntime(runtime)
}

func stopWithRuntime(rt AccessPointRuntime) error {
	zap.L().Info(fmt.Sprintf("Stopping router on %v", rt.Wlan))
	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation runtime")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP is running
	if rt.IsRunningFn == nil {
		return errors.New("missing status runtime")
	}
	running, err := rt.IsRunningFn(rt)
	if err != nil {
		return err
	}
	// stop the router
	if running {
		if rt.StopRouterFn == nil {
			return errors.New("missing router stop runtime")
		}
		if err := rt.StopRouterFn(rt); err != nil {
			return fmt.Errorf("stop router: %w", err)
		}
	} else {
		zap.L().Info("Accesspoint already stopped")
	}
	return nil
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

func isRunningWithMatcher(matcher processProbeMatcher) (bool, error) {
	out, err := runProcessProbeFn("ps", "-eo", "args").Output()
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

func isRunningWithRuntime(_ AccessPointRuntime) (bool, error) {
	matcher, err := accessPointProcessMatcher()
	if err != nil {
		return false, fmt.Errorf("build process matcher: %w", err)
	}
	return isRunningWithMatcher(matcher)
}

func checkDependencies() error {
	if _, err := exec.LookPath("hostapd"); err != nil {
		return err
	}
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		return err
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
			return "", fmt.Errorf("command %q failed with exit status %d: %v: %s", commandString, exitErr.ExitCode(), exitErr, stderrText)
		}
		return "", fmt.Errorf("command %q failed: %v: %s", commandString, err, stderrText)
	}

	// Decode the result
	zap.L().Debug(stdout.String())
	return stdout.String(), nil
}

func checkParametersWithRuntime(rt AccessPointRuntime) error {
	return checkParametersWithContext(rt)
}
