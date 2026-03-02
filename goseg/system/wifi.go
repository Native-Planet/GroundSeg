package system

import (
	"context"
	"errors"
	"fmt"
	"groundseg/structs"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hsanjuan/go-captive"
	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

var (
	proxy = &captive.Portal{
		LoginPath:           "/",
		PortalDomain:        "nativeplanet.local",
		AllowedBypassPortal: false,
		WebPath:             "c2c",
	}
	WifiInfo                    structs.SystemWifi
	Device                      string // wifi device name
	LocalUrl                    string // eg nativeplanet.local
	ConfChannel                 = make(chan string, 100)
	wifiInit                    sync.Once
	wifiStateMu                 sync.RWMutex
	c2cMu                       sync.Mutex
	c2cEnabled                                       = false
	defaultWiFiRadio            wifiRadioService     = nmcliWiFiRadioService{}
	defaultAccessPointLifecycle accessPointLifecycle = systemAccessPointLifecycle{}
	captiveAdapter                                   = newCaptiveTransportAdapter(defaultC2CServiceDeps())
	wifiInfoTickerFactory                            = func() *time.Ticker { return time.NewTicker(10 * time.Second) }
	wifiInfoLoopStopMu          sync.Mutex
	wifiInfoLoopStop            context.CancelFunc

	execCommandForWiFi   = exec.Command
	runCommandForWiFi    = runCommand
	ifCheckForWiFi       = ifCheck
	wifiNewClientForWiFi = wifi.New
)

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
	go wifiInfoLoop(ctx, device)
	return nil
}

func wifiInfoLoop(ctx context.Context, dev string) {
	ticker := wifiInfoTickerFactory()
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

func ifCheck() bool {
	out, err := runCommandForWiFi("nmcli", "radio", "wifi")
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't check interface: %v", err))
		return false
	}
	return strings.Contains(string(out), "enabled")
}

func C2CMode() error {
	zap.L().Info(fmt.Sprintf("C2C Mode initializing"))
	if err := defaultWiFiRadio.Enable(); err != nil {
		return fmt.Errorf("couldn't enable wifi interface: %w", err)
	}
	// this is necessary because it takes a while for the SSIDs to populate
	time.Sleep(10 * time.Second)
	// get wifi device
	device, err := defaultWiFiRadio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("failed to discover wifi device for C2C mode: %w", err)
	}
	// store ssids
	ssids, err := defaultWiFiRadio.ListSSIDs(device)
	if err != nil {
		return fmt.Errorf("couldn't list ssids for %s: %w", device, err)
	}
	C2CStoredSSIDs = ssids
	zap.L().Info(fmt.Sprintf("C2C retrieved available SSIDs: %v", C2CStoredSSIDs))
	// stop systemd-resolved
	_, err = runCommandForWiFi("systemctl", "stop", "systemd-resolved")
	if err != nil {
		return fmt.Errorf("failed to stop systemd-resolved: %w", err)
	}
	// stop AP
	//accesspoint.Stop(device)
	// start AP
	if err := defaultAccessPointLifecycle.Start(device); err != nil {
		return fmt.Errorf("failed to start access point: %w", err)
	}
	return nil
}

func C2CConnect(ssid, password string) error {
	zap.L().Debug("C2C Attempting to connect to ssid")
	if err := UnaliveC2C(); err != nil {
		return fmt.Errorf("disable C2C access point before wifi connect: %w", err)
	}
	device, err := defaultWiFiRadio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("discover wifi device for connect: %w", err)
	}
	if err := defaultWiFiRadio.Enable(); err != nil {
		return fmt.Errorf("start wifi device %s: %w", device, err)
	}
	time.Sleep(5 * time.Second)
	if err := defaultWiFiRadio.SetLinkUp(device); err != nil {
		return fmt.Errorf("set wifi interface %s up: %w", device, err)
	}
	// attempt to connect
	err = defaultWiFiRadio.Connect(ssid, password)
	if err != nil {
		connectErr := fmt.Errorf("connect to wifi %s: %w", ssid, err)
		if c2cErr := C2CMode(); c2cErr != nil {
			return fmt.Errorf("restore C2C mode after failed connect: %w", errors.Join(connectErr, c2cErr))
		}
		return connectErr
	} else {
		ConfChannel <- "c2cInterval"
		time.Sleep(1 * time.Second)
		//cmd := exec.Command("systemctl", "restart", "docker", "groundseg")
		cmd := execCommandForWiFi("reboot")
		_, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("reboot after C2C connect: %w", err)
		}
	}
	return nil
}

func UnaliveC2C() error {
	// stop AP
	device, err := defaultWiFiRadio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("failed to discover wifi device for C2C shutdown: %w", err)
	}
	if err := defaultAccessPointLifecycle.Stop(device); err != nil {
		return fmt.Errorf("failed to stop access point on %s: %w", device, err)
	}
	// start systemd-resolved
	return EnableResolved()
}

func EnableResolved() error {
	cmd := execCommandForWiFi("systemctl", "enable", "systemd-resolved")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable systemd-resolved: %w", err)
	}
	cmd = execCommandForWiFi("systemctl", "start", "systemd-resolved")
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start systemd-resolved: %w", err)
	}
	return nil
}

func CaptivePortal(dev string) error {
	return captiveAdapter.runPortal(proxy)
}

func CaptiveAPI(w http.ResponseWriter, r *http.Request) {
	captiveAdapter.handleAPI(w, r)
}

func announceNetworks(dev string) {
	captiveAdapter.broadcastNetworks(dev)
}

func getWifiDevice() ([]string, error) {
	cmd := "nmcli device status | grep wifi | awk '{print $1}'"
	out, err := execCommandForWiFi("sh", "-c", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("no WiFi device found: %w", err)
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
		return nil, fmt.Errorf("no WiFi device found")
	}
	return wifiDevices, nil
}

func primaryWifiDevice() (string, error) {
	wifiDevices, err := getWifiDevice()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve wifi devices: %w", err)
	}
	if len(wifiDevices) == 0 {
		return "", fmt.Errorf("no WiFi device found")
	}
	return wifiDevices[0], nil
}

func ListWifiSSIDs(dev string) ([]string, error) {
	out, err := runCommandForWiFi("nmcli", "-t", "dev", "wifi", "list", "ifname", dev)
	if err != nil {
		return nil, fmt.Errorf("couldn't gather wifi networks: %w", err)
	}
	lines := strings.Split(out, "\n")
	var ssids []string
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) > 7 && parts[7] != "" {
			ssids = append(ssids, parts[7])
		}
	}
	return ssids, nil
}

func getConnectedSSID(c *wifi.Client, dev string) string {
	interfaces, err := c.Interfaces()
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get devices: %v", err))
		return ""
	}
	for _, iface := range interfaces {
		if iface.Name == dev && iface.Type == wifi.InterfaceTypeStation {
			bss, err := c.BSS(iface)
			if err != nil {
				continue
			}
			return bss.SSID
		}
	}
	return ""
}

func ConnectToWifi(ssid, password string) error {
	// Connect to WiFi using nmcli
	cmd := execCommandForWiFi("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %w", err)
	}
	return nil
}

func DisconnectWifi(ifaceName string) error {
	c, err := wifiNewClientForWiFi()
	if err != nil {
		return fmt.Errorf("couldn't create wifi client: %w", err)
	}
	defer c.Close()
	iface := &wifi.Interface{Name: ifaceName}
	if err := c.Disconnect(iface); err != nil {
		return fmt.Errorf("failed to disconnect wifi interface %s: %w", ifaceName, err)
	}
	return nil
}

func ToggleDevice(dev string) error {
	var cmd string
	if ifCheckForWiFi() {
		cmd = "off"
	} else {
		cmd = "on"
	}
	_, err := runCommandForWiFi("nmcli", "radio", "wifi", cmd)
	if err != nil {
		return fmt.Errorf("failed to set wifi radio %s: %w", cmd, err)
	}
	return nil
}

func applyCaptiveRules(dev string) error {
	sysctlSettings := map[string]string{
		"net.ipv4.ip_forward":              "1",
		"net.ipv6.conf.all.forwarding":     "1",
		"net.ipv4.conf.all.send_redirects": "0",
	}
	for key, value := range sysctlSettings {
		if _, err := runCommandForWiFi("sysctl", "-w", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}
