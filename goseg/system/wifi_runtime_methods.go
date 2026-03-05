package system

import (
	"errors"
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

func (runtime wifiRuntime) isWiFiRadioEnabled() (bool, error) {
	out, err := runtime.runNmcli("radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false, fmt.Errorf("couldn't check interface: %w", err)
	}
	return strings.Contains(out, "enabled"), nil
}

func (runtime wifiRuntime) wifiDevices() ([]string, error) {
	devices, err := runtime.wifiDevicesBestEffort()
	if errors.Is(err, ErrWiFiPartialResult) {
		return nil, fmt.Errorf("list wifi devices strictly: %w", err)
	}
	return devices, err
}

func (runtime wifiRuntime) wifiDevicesBestEffort() ([]string, error) {
	out, err := runtime.runNmcli("-t", "-f", "DEVICE,TYPE", "device", "status")
	if err != nil {
		return nil, fmt.Errorf("couldn't read wifi devices: %w", err)
	}
	devices, parseErr := parseNmcliWifiDevices(out)
	if parseErr == nil {
		return devices, nil
	}
	wrappedErr := wrapWiFiPartialResult("list wifi devices", parseErr)
	if errors.Is(wrappedErr, ErrWiFiPartialResult) && len(devices) > 0 {
		return devices, wrappedErr
	}
	return nil, wrappedErr
}

func (runtime wifiRuntime) primaryWifiDevice() (string, error) {
	devices, err := runtime.wifiDevicesBestEffort()
	if err != nil {
		if !errors.Is(err, ErrWiFiPartialResult) || len(devices) == 0 {
			return "", fmt.Errorf("failed to retrieve wifi devices: %w", err)
		}
		return devices[0], err
	}
	if len(devices) == 0 {
		return "", ErrWiFiInterfaceNotFound
	}
	return devices[0], nil
}

func (runtime wifiRuntime) listSSIDs(interfaceName string) ([]string, error) {
	ssids, err := runtime.listSSIDsBestEffort(interfaceName)
	if errors.Is(err, ErrWiFiPartialResult) {
		return nil, fmt.Errorf("list wifi ssids strictly: %w", err)
	}
	return ssids, err
}

func (runtime wifiRuntime) listSSIDsBestEffort(interfaceName string) ([]string, error) {
	out, err := runtime.runNmcli("-t", "dev", "wifi", "list", "ifname", interfaceName)
	if err != nil {
		return nil, fmt.Errorf("couldn't gather wifi networks: %w", err)
	}
	ssids, parseErr := parseNmcliWifiSSIDs(out)
	if parseErr == nil {
		return ssids, nil
	}
	wrappedErr := wrapWiFiPartialResult("list wifi SSIDs", parseErr)
	if errors.Is(wrappedErr, ErrWiFiPartialResult) && len(ssids) > 0 {
		return ssids, wrappedErr
	}
	return nil, wrappedErr
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

func (runtime wifiRuntime) resolveToggleTarget(interfaceName string) (string, error) {
	target := strings.TrimSpace(interfaceName)
	if target != "" {
		return target, nil
	}
	resolvedTarget, resolveErr := runtime.primaryWifiDevice()
	if resolveErr == nil {
		return resolvedTarget, nil
	}
	if !errors.Is(resolveErr, ErrWiFiPartialResult) || resolvedTarget == "" {
		return "", fmt.Errorf("resolve wifi device for toggle: %w", resolveErr)
	}
	zap.L().Warn(resolveErr.Error())
	return resolvedTarget, nil
}

func (runtime wifiRuntime) deriveToggleCommand() (string, error) {
	wifiEnabled, err := runtime.isWiFiRadioEnabled()
	if err != nil {
		return "", fmt.Errorf("failed to detect wifi radio state: %w", err)
	}
	if wifiEnabled {
		return "down", nil
	}
	return "up", nil
}

func (runtime wifiRuntime) executeLinkToggle(target, command string) error {
	_, err := runtime.RunCommand("ip", "link", "set", target, command)
	if err != nil {
		return fmt.Errorf("failed to toggle wifi interface %s %s: %w", target, command, err)
	}
	return nil
}

func (runtime wifiRuntime) toggleDevice(interfaceName string) error {
	target, err := runtime.resolveToggleTarget(interfaceName)
	if err != nil {
		return err
	}
	command, err := runtime.deriveToggleCommand()
	if err != nil {
		return err
	}
	return runtime.executeLinkToggle(target, command)
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
