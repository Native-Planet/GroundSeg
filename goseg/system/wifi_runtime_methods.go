package system

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

type wifiRuntime struct {
	execCommand        func(string, ...string) *exec.Cmd
	runCommand         func(string, ...string) (string, error)
	newWifiClient      func() (*wifi.Client, error)
	clientInterfacesFn func(*wifi.Client) ([]*wifi.Interface, error)
	clientBSSFn        func(*wifi.Client, *wifi.Interface) (*wifi.BSS, error)
	wifiInfoTicker     func() *time.Ticker
}

func NewWiFiRuntime() wifiRuntime {
	return wifiRuntime{
		execCommand:        exec.Command,
		runCommand:         runCommand,
		newWifiClient:      wifi.New,
		clientInterfacesFn: func(c *wifi.Client) ([]*wifi.Interface, error) { return c.Interfaces() },
		clientBSSFn:        func(c *wifi.Client, iface *wifi.Interface) (*wifi.BSS, error) { return c.BSS(iface) },
		wifiInfoTicker:     func() *time.Ticker { return time.NewTicker(10 * time.Second) },
	}
}

func DefaultWiFiRuntime() wifiRuntime {
	return defaultWiFiRuntimeValue
}

func defaultWiFiRuntime() wifiRuntime {
	return DefaultWiFiRuntime()
}

func withWiFiRuntime(overrides wifiRuntime) wifiRuntime {
	return NewWiFiRuntimeWith(overrides)
}

func NewWiFiRuntimeWith(overrides wifiRuntime) wifiRuntime {
	return mergeWiFiRuntime(defaultWiFiRuntime(), overrides)
}

func mergeWiFiRuntime(defaults, overrides wifiRuntime) wifiRuntime {
	if overrides.execCommand != nil {
		defaults.execCommand = overrides.execCommand
	}
	if overrides.runCommand != nil {
		defaults.runCommand = overrides.runCommand
	}
	if overrides.newWifiClient != nil {
		defaults.newWifiClient = overrides.newWifiClient
	}
	if overrides.clientInterfacesFn != nil {
		defaults.clientInterfacesFn = overrides.clientInterfacesFn
	}
	if overrides.clientBSSFn != nil {
		defaults.clientBSSFn = overrides.clientBSSFn
	}
	if overrides.wifiInfoTicker != nil {
		defaults.wifiInfoTicker = overrides.wifiInfoTicker
	}
	return defaults
}

func (runtime wifiRuntime) ifCheck() (bool, error) {
	out, err := runtime.runCommand("nmcli", "radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false, fmt.Errorf("couldn't check interface: %w", err)
	}
	return strings.Contains(out, "enabled"), nil
}

func (runtime wifiRuntime) wifiDevices() ([]string, error) {
	cmd := "nmcli device status | grep wifi | awk '{print $1}'"
	out, err := runtime.execCommand("sh", "-c", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("couldn't read wifi devices: %w", err)
	}
	rawDevices := strings.Split(strings.TrimSpace(string(out)), "\n")
	wifiDevices := make([]string, 0, len(rawDevices))
	for _, device := range rawDevices {
		trimmed := strings.TrimSpace(device)
		if trimmed != "" {
			wifiDevices = append(wifiDevices, trimmed)
		}
	}
	if len(wifiDevices) == 0 {
		return nil, ErrWifiInterfaceNotFound
	}
	return wifiDevices, nil
}

func (runtime wifiRuntime) primaryWifiDevice() (string, error) {
	devices, err := runtime.wifiDevices()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve wifi devices: %w", err)
	}
	if len(devices) == 0 {
		return "", ErrWifiInterfaceNotFound
	}
	return devices[0], nil
}

func (runtime wifiRuntime) listSSIDs(dev string) ([]string, error) {
	out, err := runtime.runCommand("nmcli", "-t", "dev", "wifi", "list", "ifname", dev)
	if err != nil {
		return nil, fmt.Errorf("couldn't gather wifi networks: %w", err)
	}
	lines := strings.Split(out, "\n")
	ssids := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) > 7 && parts[7] != "" {
			ssids = append(ssids, parts[7])
		}
	}
	return ssids, nil
}

func (runtime wifiRuntime) connectedSSID(client *wifi.Client, dev string) (string, error) {
	interfaces, err := runtime.clientInterfacesFn(client)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get devices: %v", err))
		return "", fmt.Errorf("couldn't get devices: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name == dev && iface.Type == wifi.InterfaceTypeStation {
			bss, err := runtime.clientBSSFn(client, iface)
			if err != nil {
				return "", fmt.Errorf("failed to get BSS for %s: %w", dev, err)
			}
			return bss.SSID, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrWifiInterfaceNotFound, dev)
}

func (runtime wifiRuntime) connect(ssid, password string) error {
	cmd := runtime.execCommand("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %w", err)
	}
	return nil
}

func (runtime wifiRuntime) disconnect(ifaceName string) error {
	client, err := runtime.newWifiClient()
	if err != nil {
		return fmt.Errorf("couldn't create wifi client: %w", err)
	}
	defer client.Close()
	iface := &wifi.Interface{Name: ifaceName}
	if err := client.Disconnect(iface); err != nil {
		return fmt.Errorf("failed to disconnect wifi interface %s: %w", ifaceName, err)
	}
	return nil
}

func (runtime wifiRuntime) toggleDevice(dev string) error {
	_ = dev
	wifiEnabled, err := runtime.ifCheck()
	if err != nil {
		return fmt.Errorf("failed to detect wifi radio state: %w", err)
	}
	cmd := "off"
	if !wifiEnabled {
		cmd = "on"
	}
	_, err = runtime.runCommand("nmcli", "radio", "wifi", cmd)
	if err != nil {
		return fmt.Errorf("failed to set wifi radio %s: %w", cmd, err)
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
		if _, err := runtime.runCommand("sysctl", "-w", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}
