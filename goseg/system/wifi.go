package system

import (
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

	execCommandForWiFi   = exec.Command
	runCommandForWiFi    = runCommand
	ifCheckForWiFi       = ifCheck
	wifiNewClientForWiFi = wifi.New
)

func InitializeWiFi() error {
	var initErr error
	wifiInit.Do(func() {
		device, err := defaultWiFiRadio.PrimaryDevice()
		if err != nil {
			initErr = fmt.Errorf("couldn't find a wifi device: %w", err)
			return
		}
		Device = device
		defaultWiFiRadio.RefreshInfo(device)
		go wifiInfoLoop(device)
	})
	return initErr
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

func wifiInfoLoop(dev string) {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
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
		zap.L().Error(fmt.Sprintf("Couldn't check interface: %v", err))
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
		return fmt.Errorf("Failed to stop resolved: %w", err)
	}
	// stop AP
	//accesspoint.Stop(device)
	// start AP
	if err := defaultAccessPointLifecycle.Start(device); err != nil {
		return fmt.Errorf("Failed to start accesspoint: %w", err)
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
		return err
	}
	// attempt to connect
	err = defaultWiFiRadio.Connect(ssid, password)
	if err != nil {
		connectErr := fmt.Errorf("connect to wifi %s: %w", ssid, err)
		if c2cErr := C2CMode(); c2cErr != nil {
			return fmt.Errorf("%w; restore C2C mode after failed connect: %v", connectErr, c2cErr)
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
		return fmt.Errorf("Failed to enable systemd-resolved: %v", err)
	}
	cmd = execCommandForWiFi("systemctl", "start", "systemd-resolved")
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start systemd-resolved: %v", err)
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
		return "", err
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
		zap.L().Error(fmt.Sprintf("Couldn't get devices: %v", err))
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
		return fmt.Errorf("failed to connect to wifi: %v", err)
	}
	return nil
}

func DisconnectWifi(ifaceName string) error {
	c, err := wifiNewClientForWiFi()
	if err != nil {
		return err
	}
	defer c.Close()
	iface := &wifi.Interface{Name: ifaceName}
	return c.Disconnect(iface)
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
		return err
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
			return fmt.Errorf("failed to set %s: %v", key, err)
		}
	}
	return nil
}
