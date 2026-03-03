package system

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
	"groundseg/internal/seams"
)

type wifiRuntime struct {
	ExecCommand        func(string, ...string) *exec.Cmd
	RunCommand         func(string, ...string) (string, error)
	NewWifiClient      func() (*wifi.Client, error)
	ClientInterfacesFn func(*wifi.Client) ([]*wifi.Interface, error)
	ClientBSSFn        func(*wifi.Client, *wifi.Interface) (*wifi.BSS, error)
	WifiInfoTicker     func() *time.Ticker
}

func newWiFiRuntime() wifiRuntime {
	return wifiRuntime{
		ExecCommand:        exec.Command,
		RunCommand:         runCommand,
		NewWifiClient:      wifi.New,
		ClientInterfacesFn: func(c *wifi.Client) ([]*wifi.Interface, error) { return c.Interfaces() },
		ClientBSSFn:        func(c *wifi.Client, iface *wifi.Interface) (*wifi.BSS, error) { return c.BSS(iface) },
		WifiInfoTicker:     func() *time.Ticker { return time.NewTicker(10 * time.Second) },
	}
}

func NewWiFiRuntime() wifiRuntime {
	return newWiFiRuntime()
}

func DefaultWiFiRuntime() wifiRuntime {
	return newWiFiRuntime()
}

func NewWiFiRuntimeWith(overrides wifiRuntime) wifiRuntime {
	return seams.MergeAll(newWiFiRuntime(), overrides)
}

func (runtime wifiRuntime) ifCheck() (bool, error) {
	out, err := runtime.RunCommand("nmcli", "radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false, fmt.Errorf("couldn't check interface: %w", err)
	}
	return strings.Contains(out, "enabled"), nil
}

func (runtime wifiRuntime) wifiDevices() ([]string, error) {
	cmd := "nmcli device status | grep wifi | awk '{print $1}'"
	out, err := runtime.ExecCommand("sh", "-c", cmd).Output()
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
	out, err := runtime.RunCommand("nmcli", "-t", "dev", "wifi", "list", "ifname", dev)
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
	interfaces, err := runtime.ClientInterfacesFn(client)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get devices: %v", err))
		return "", fmt.Errorf("couldn't get devices: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name == dev && iface.Type == wifi.InterfaceTypeStation {
			bss, err := runtime.ClientBSSFn(client, iface)
			if err != nil {
				return "", fmt.Errorf("failed to get BSS for %s: %w", dev, err)
			}
			return bss.SSID, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrWifiInterfaceNotFound, dev)
}

func (runtime wifiRuntime) connect(ssid, password string) error {
	cmd := runtime.ExecCommand("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %w", err)
	}
	return nil
}

func (runtime wifiRuntime) disconnect(ifaceName string) error {
	client, err := runtime.NewWifiClient()
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
	target := strings.TrimSpace(dev)
	if target == "" {
		var err error
		target, err = runtime.primaryWifiDevice()
		if err != nil {
			return fmt.Errorf("resolve wifi device for toggle: %w", err)
		}
	}

	wifiEnabled, err := runtime.ifCheck()
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
