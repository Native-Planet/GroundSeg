package system

import (
	"context"
	"errors"
	"fmt"
	"groundseg/structs"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

var (
	WifiInfo           structs.SystemWifi
	Device             string // wifi device name
	LocalUrl           string // eg nativeplanet.local
	ConfChannel        = make(chan string, 100)
	wifiInit           sync.Once
	wifiStateMu        sync.RWMutex
	c2cMu              sync.Mutex
	c2cEnabled                          = false
	defaultWiFiRadio   wifiRadioService = nmcliWiFiRadioService{}
	defaultWiFiService                  = newWiFiService(wifiRuntime{})
	wifiInfoLoopStopMu sync.Mutex
	wifiInfoLoopStop   context.CancelFunc
)

type wifiRuntime struct {
	execCommand        func(string, ...string) *exec.Cmd
	runCommand         func(string, ...string) (string, error)
	newWifiClient      func() (*wifi.Client, error)
	clientInterfacesFn func(*wifi.Client) ([]*wifi.Interface, error)
	clientBSSFn        func(*wifi.Client, *wifi.Interface) (*wifi.BSS, error)
	wifiInfoTicker     func() *time.Ticker
}

func defaultWiFiRuntime() wifiRuntime {
	return wifiRuntime{
		execCommand:        exec.Command,
		runCommand:         runCommand,
		newWifiClient:      wifi.New,
		clientInterfacesFn: func(c *wifi.Client) ([]*wifi.Interface, error) { return c.Interfaces() },
		clientBSSFn:        func(c *wifi.Client, iface *wifi.Interface) (*wifi.BSS, error) { return c.BSS(iface) },
		wifiInfoTicker:     func() *time.Ticker { return time.NewTicker(10 * time.Second) },
	}
}

func withWiFiRuntime(overrides wifiRuntime) wifiRuntime {
	runtime := defaultWiFiRuntime()
	if overrides.execCommand != nil {
		runtime.execCommand = overrides.execCommand
	}
	if overrides.runCommand != nil {
		runtime.runCommand = overrides.runCommand
	}
	if overrides.newWifiClient != nil {
		runtime.newWifiClient = overrides.newWifiClient
	}
	if overrides.clientInterfacesFn != nil {
		runtime.clientInterfacesFn = overrides.clientInterfacesFn
	}
	if overrides.clientBSSFn != nil {
		runtime.clientBSSFn = overrides.clientBSSFn
	}
	if overrides.wifiInfoTicker != nil {
		runtime.wifiInfoTicker = overrides.wifiInfoTicker
	}
	return runtime
}

type wifiOperations struct {
	runtime wifiRuntime
}

func newWiFiService(runtime wifiRuntime) wifiOperations {
	return wifiOperations{
		runtime: withWiFiRuntime(runtime),
	}
}

func (svc wifiOperations) ifCheck() (bool, error) {
	out, err := svc.runtime.runCommand("nmcli", "radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false, fmt.Errorf("couldn't check interface: %w", err)
	}
	return strings.Contains(out, "enabled"), nil
}

func (svc wifiOperations) wifiDevices() ([]string, error) {
	cmd := "nmcli device status | grep wifi | awk '{print $1}'"
	out, err := svc.runtime.execCommand("sh", "-c", cmd).Output()
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

func (svc wifiOperations) primaryWifiDevice() (string, error) {
	devices, err := svc.wifiDevices()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve wifi devices: %w", err)
	}
	if len(devices) == 0 {
		return "", ErrWifiInterfaceNotFound
	}
	return devices[0], nil
}

func (svc wifiOperations) listSSIDs(dev string) ([]string, error) {
	out, err := svc.runtime.runCommand("nmcli", "-t", "dev", "wifi", "list", "ifname", dev)
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

func (svc wifiOperations) connectedSSID(client *wifi.Client, dev string) (string, error) {
	interfaces, err := svc.runtime.clientInterfacesFn(client)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get devices: %v", err))
		return "", fmt.Errorf("couldn't get devices: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name == dev && iface.Type == wifi.InterfaceTypeStation {
			bss, err := svc.runtime.clientBSSFn(client, iface)
			if err != nil {
				return "", fmt.Errorf("failed to get BSS for %s: %w", dev, err)
			}
			return bss.SSID, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrWifiInterfaceNotFound, dev)
}

func (svc wifiOperations) connect(ssid, password string) error {
	cmd := svc.runtime.execCommand("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %w", err)
	}
	return nil
}

func (svc wifiOperations) disconnect(ifaceName string) error {
	client, err := svc.runtime.newWifiClient()
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

func (svc wifiOperations) toggleDevice(dev string) error {
	_ = dev
	wifiEnabled, err := svc.ifCheck()
	if err != nil {
		return fmt.Errorf("failed to detect wifi radio state: %w", err)
	}
	cmd := "off"
	if !wifiEnabled {
		cmd = "on"
	}
	_, err = svc.runtime.runCommand("nmcli", "radio", "wifi", cmd)
	if err != nil {
		return fmt.Errorf("failed to set wifi radio %s: %w", cmd, err)
	}
	return nil
}

func (svc wifiOperations) applyCaptiveRules(dev string) error {
	_ = dev
	sysctlSettings := map[string]string{
		"net.ipv4.ip_forward":              "1",
		"net.ipv6.conf.all.forwarding":     "1",
		"net.ipv4.conf.all.send_redirects": "0",
	}
	for key, value := range sysctlSettings {
		if _, err := svc.runtime.runCommand("sysctl", "-w", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}

var ErrWifiInterfaceNotFound = errors.New("wifi interface not found")

func HasWifiDevice() bool {
	wifiStateMu.RLock()
	defer wifiStateMu.RUnlock()
	return Device != ""
}

func InitializeWiFi() error {
	var initErr error
	wifiInit.Do(func() {
		initErr = startWiFiInfoLoop(context.Background())
	})
	return initErr
}

func StartWiFiInfoLoop(ctx context.Context) error {
	return startWiFiInfoLoop(ctx)
}

func StopWiFiInfoLoop() {
	wifiInfoLoopStopMu.Lock()
	stop := wifiInfoLoopStop
	wifiInfoLoopStop = nil
	wifiInfoLoopStopMu.Unlock()
	if stop != nil {
		stop()
	}
}

func IsC2CMode() bool {
	c2cMu.Lock()
	defer c2cMu.Unlock()
	return c2cEnabled
}

func SetC2CMode(isTrue bool) error {
	c2cMu.Lock()
	defer c2cMu.Unlock()
	c2cEnabled = isTrue
	return nil
}

func setWifiInfo(info structs.SystemWifi) {
	wifiStateMu.Lock()
	defer wifiStateMu.Unlock()
	WifiInfo = info
}

func WifiInfoSnapshot() structs.SystemWifi {
	wifiStateMu.RLock()
	defer wifiStateMu.RUnlock()
	return WifiInfo
}

func startWiFiInfoLoop(ctx context.Context) error {
	device, err := defaultWiFiRadio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("couldn't find a wifi device: %w", err)
	}
	{
		wifiStateMu.Lock()
		Device = device
		wifiStateMu.Unlock()
	}
	defaultWiFiRadio.RefreshInfo(device)

	wifiInfoLoopStopMu.Lock()
	if wifiInfoLoopStop != nil {
		wifiInfoLoopStopMu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	wifiInfoLoopStop = cancel
	wifiInfoLoopStopMu.Unlock()
	go wifiInfoLoop(ctx, device, defaultWiFiService.runtime)
	return nil
}

func wifiInfoLoop(ctx context.Context, dev string, runtime wifiRuntime) {
	svc := newWiFiService(runtime)
	ticker := svc.runtime.wifiInfoTicker()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			defaultWiFiRadio.RefreshInfo(dev)
		}
	}
}

func constructWifiInfo(dev string) {
	defaultWiFiRadio.RefreshInfo(dev)
}

func ListWifiSSIDs(dev string) ([]string, error) {
	return defaultWiFiService.listSSIDs(dev)
}

func ConnectToWifi(ssid, password string) error {
	return defaultWiFiService.connect(ssid, password)
}

func DisconnectWifi(ifaceName string) error {
	return defaultWiFiService.disconnect(ifaceName)
}

func ToggleDevice(dev string) error {
	return defaultWiFiService.toggleDevice(dev)
}

func IsWifiEnabled() (bool, error) {
	return defaultWiFiService.ifCheck()
}

func applyCaptiveRules(dev string) error {
	return defaultWiFiService.applyCaptiveRules(dev)
}
