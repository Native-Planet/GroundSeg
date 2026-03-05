package system

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"groundseg/internal/resource"
	"groundseg/internal/seams"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

type wifiRuntime struct {
	ExecCommand        func(string, ...string) *exec.Cmd
	RunCommand         func(string, ...string) (string, error)
	RunNmcliFn         func(args ...string) (string, error)
	NewWifiClient      func() (*wifi.Client, error)
	ClientInterfacesFn func(*wifi.Client) ([]*wifi.Interface, error)
	ClientBSSFn        func(*wifi.Client, *wifi.Interface) (*wifi.BSS, error)
	WifiInfoTicker     func() *time.Ticker
}

type wifiNmcliDevice struct {
	name string
	typ  string
}

type wifiNmcliScanResult struct {
	ssid string
}

type wifiNmcliParseRecord struct {
	index int
	line  string
	data  []string
}

type wifiNmcliParseError struct {
	record    wifiNmcliParseRecord
	required  int
	found     int
	operation string
}

func (e wifiNmcliParseError) Error() string {
	return "invalid nmcli output for " + e.operation + "; expected " +
		fmt.Sprintf("%d fields", e.required) + ", found " + fmt.Sprintf("%d", e.found)
}

func newWiFiNmcliParseError(operation, line string, index, required, found int) wifiNmcliParseError {
	return wifiNmcliParseError{
		record: wifiNmcliParseRecord{
			index: index,
			line:  line,
			data:  splitNmcliRecord(line),
		},
		required:  required,
		found:     found,
		operation: operation,
	}
}

func splitNmcliRecord(raw string) []string {
	fields := make([]string, 0, 8)
	var field strings.Builder
	escaped := false
	for _, char := range raw {
		switch {
		case escaped:
			field.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == ':':
			fields = append(fields, strings.TrimSpace(field.String()))
			field.Reset()
		default:
			field.WriteRune(char)
		}
	}
	fields = append(fields, strings.TrimSpace(field.String()))
	return fields
}

func parseNmcliWifiDeviceRecord(raw string, index int) (wifiNmcliDevice, error) {
	fields := splitNmcliRecord(raw)
	if len(fields) < 2 {
		return wifiNmcliDevice{}, newWiFiNmcliParseError(
			"wifi device list", raw, index, 2, len(fields),
		)
	}
	return wifiNmcliDevice{name: strings.TrimSpace(fields[0]), typ: strings.TrimSpace(fields[1])}, nil
}

func parseNmcliWifiScanResultRecord(raw string, index int) (wifiNmcliScanResult, error) {
	fields := splitNmcliRecord(raw)
	if len(fields) <= 7 {
		return wifiNmcliScanResult{}, newWiFiNmcliParseError(
			"wifi scan result", raw, index, 8, len(fields),
		)
	}
	return wifiNmcliScanResult{ssid: strings.TrimSpace(fields[7])}, nil
}

func parseNmcliWifiDeviceRecords(raw string) ([]string, error) {
	rawLines := strings.Split(strings.TrimSpace(raw), "\n")
	devices := make([]string, 0, len(rawLines))
	parseErrors := make([]error, 0)
	for index, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		deviceRecord, err := parseNmcliWifiDeviceRecord(trimmed, index)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		if deviceRecord.typ == "wifi" && deviceRecord.name != "" {
			devices = append(devices, deviceRecord.name)
		}
	}
	if len(parseErrors) > 0 && len(devices) == 0 {
		return nil, parseErrors[0]
	}
	return devices, nil
}

func parseNmcliWifiScanResults(raw string) ([]wifiNmcliScanResult, error) {
	rawLines := strings.Split(strings.TrimSpace(raw), "\n")
	results := make([]wifiNmcliScanResult, 0, len(rawLines))
	parseErrors := make([]error, 0)
	for index, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		record, err := parseNmcliWifiScanResultRecord(trimmed, index)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		if record.ssid == "" {
			continue
		}
		results = append(results, record)
	}
	if len(results) == 0 && len(parseErrors) > 0 {
		return nil, parseErrors[0]
	}
	return results, nil
}

func NewWiFiRuntime() wifiRuntime {
	return wifiRuntime{
		ExecCommand: exec.Command,
		RunCommand:  runCommand,
		RunNmcliFn: func(args ...string) (string, error) {
			return runCommand("nmcli", args...)
		},
		NewWifiClient:      wifi.New,
		ClientInterfacesFn: func(c *wifi.Client) ([]*wifi.Interface, error) { return c.Interfaces() },
		ClientBSSFn:        func(c *wifi.Client, iface *wifi.Interface) (*wifi.BSS, error) { return c.BSS(iface) },
		WifiInfoTicker:     func() *time.Ticker { return time.NewTicker(10 * time.Second) },
	}
}

func NewWiFiRuntimeWith(overrides wifiRuntime) wifiRuntime {
	runtime := seams.MergeAll(NewWiFiRuntime(), overrides)
	if overrides.RunNmcliFn == nil {
		runCommand := runtime.RunCommand
		runtime.RunNmcliFn = func(args ...string) (string, error) {
			return runCommand("nmcli", args...)
		}
	}
	return runtime
}

func (runtime wifiRuntime) runNmcli(args ...string) (string, error) {
	if runtime.RunNmcliFn != nil {
		return runtime.RunNmcliFn(args...)
	}
	return runtime.RunCommand("nmcli", args...)
}

func parseNmcliWifiDevices(raw string) ([]string, error) {
	parsedDevices, err := parseNmcliWifiDeviceRecords(raw)
	if err != nil {
		return nil, err
	}
	devices := make([]string, 0, len(parsedDevices))
	devices = append(devices, parsedDevices...)
	if len(devices) == 0 {
		return nil, ErrWiFiInterfaceNotFound
	}
	return devices, nil
}

func parseNmcliWifiSSIDs(raw string) ([]string, error) {
	scans, err := parseNmcliWifiScanResults(raw)
	if err != nil {
		return nil, err
	}
	ssids := make([]string, 0, len(scans))
	for _, scan := range scans {
		ssids = append(ssids, scan.ssid)
	}
	return ssids, nil
}

func (runtime wifiRuntime) isWiFiRadioEnabled() (bool, error) {
	out, err := runtime.runNmcli("radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false, fmt.Errorf("couldn't check interface: %w", err)
	}
	return strings.Contains(out, "enabled"), nil
}

func (runtime wifiRuntime) wifiDevices() ([]string, error) {
	out, err := runtime.runNmcli("-t", "-f", "DEVICE,TYPE", "device", "status")
	if err != nil {
		return nil, fmt.Errorf("couldn't read wifi devices: %w", err)
	}
	return parseNmcliWifiDevices(out)
}

func (runtime wifiRuntime) primaryWifiDevice() (string, error) {
	devices, err := runtime.wifiDevices()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve wifi devices: %w", err)
	}
	if len(devices) == 0 {
		return "", ErrWiFiInterfaceNotFound
	}
	return devices[0], nil
}

func (runtime wifiRuntime) listSSIDs(interfaceName string) ([]string, error) {
	out, err := runtime.runNmcli("-t", "dev", "wifi", "list", "ifname", interfaceName)
	if err != nil {
		return nil, fmt.Errorf("couldn't gather wifi networks: %w", err)
	}
	return parseNmcliWifiSSIDs(out)
}

func (runtime wifiRuntime) connectedSSID(client *wifi.Client, interfaceName string) (string, error) {
	interfaces, err := runtime.ClientInterfacesFn(client)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get devices: %v", err))
		return "", fmt.Errorf("couldn't get devices: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name == interfaceName && iface.Type == wifi.InterfaceTypeStation {
			bss, err := runtime.ClientBSSFn(client, iface)
			if err != nil {
				return "", fmt.Errorf("failed to get BSS for %s: %w", interfaceName, err)
			}
			return bss.SSID, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrWiFiInterfaceNotFound, interfaceName)
}

func (runtime wifiRuntime) connect(ssid, password string) error {
	cmd := runtime.ExecCommand("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %w", err)
	}
	return nil
}

func (runtime wifiRuntime) disconnect(interfaceName string) (err error) {
	client, err := runtime.NewWifiClient()
	if err != nil {
		return fmt.Errorf("couldn't create wifi client: %w", err)
	}
	defer func() {
		err = resource.JoinCloseError(err, client, "disconnect from wifi")
	}()
	iface := &wifi.Interface{Name: interfaceName}
	if err := client.Disconnect(iface); err != nil {
		return fmt.Errorf("failed to disconnect wifi interface %s: %w", interfaceName, err)
	}
	return nil
}

func (runtime wifiRuntime) toggleDevice(interfaceName string) error {
	target := strings.TrimSpace(interfaceName)
	if target == "" {
		var err error
		target, err = runtime.primaryWifiDevice()
		if err != nil {
			return fmt.Errorf("resolve wifi device for toggle: %w", err)
		}
	}

	wifiEnabled, err := runtime.isWiFiRadioEnabled()
	if err != nil {
		return fmt.Errorf("failed to detect wifi radio state: %w", err)
	}

	command := "down"
	if !wifiEnabled {
		command = "up"
	}
	_, err = runtime.RunCommand("ip", "link", "set", target, command)
	if err != nil {
		return fmt.Errorf("failed to toggle wifi interface %s %s: %w", target, command, err)
	}
	return nil
}

func (runtime wifiRuntime) applyCaptiveRules() error {
	sysctlSettings := map[string]string{
		"net.ipv4.ip_forward":              "1",
		"net.ipv6.conf.all.forwarding":     "1",
		"net.ipv4.conf.all.send_redirects": "0",
	}
	for key, value := range sysctlSettings {
		if _, err := runtime.RunCommand("sysctl", "-w", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}
