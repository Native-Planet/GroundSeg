package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goseg/accesspoint"
	"goseg/logger"
	"goseg/structs"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hsanjuan/go-captive"
	"github.com/mdlayher/wifi"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients = make(map[*websocket.Conn]bool)
	proxy   = &captive.Portal{
		LoginPath:           "/",
		PortalDomain:        "nativeplanet.local",
		AllowedBypassPortal: false,
		WebPath:             "c2c",
	}
	WifiInfo structs.SystemWifi
	Device   string // wifi device name
	LocalUrl string // eg nativeplanet.local

	c2cEnabled = false
	c2cMu      sync.Mutex
)

func init() {
	dev, err := getWifiDevice()
	if err != nil || dev[0] == "" {
		logger.Logger.Error(fmt.Sprintf("Couldn't find a wifi device! %v", err))
		return
	} else {
		Device = dev[0]
		constructWifiInfo(dev[0])
		go wifiInfoLoop(dev[0])
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

func wifiInfoLoop(dev string) {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			constructWifiInfo(dev)
		}
	}
}

func constructWifiInfo(dev string) {
	WifiInfo.Status = ifCheck()
	if WifiInfo.Status {
		c, err := wifi.New()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't create wifi client with device %v: %v", dev, err))
			WifiInfo.Status = false
			WifiInfo.Active = ""
			WifiInfo.Networks = []string{}
		}
		defer c.Close()
		active := getConnectedSSID(c, dev)
		WifiInfo.Active = active
		WifiInfo.Networks = ListWifiSSIDs(dev)
	} else {
		WifiInfo.Active = ""
		WifiInfo.Networks = []string{}
	}
}

func ifCheck() bool {
	out, err := runCommand("nmcli", "radio", "wifi")
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't check interface: %v", err))
		return false
	}
	return strings.Contains(string(out), "enabled")
}

func C2CMode() error {
	logger.Logger.Debug(fmt.Sprintf("C2C Mode called"))
	// make sure wifi is enabled
	runCommand("nmcli", "radio", "wifi", "on")
	// this is necessary because it takes a while for the SSIDs to populate
	time.Sleep(10 * time.Second)
	// get wifi device
	dev, _ := getWifiDevice()
	// todo: start wifi if not started
	// store ssids
	C2CStoredSSIDs = ListWifiSSIDs(dev[0])

	// stop systemd-resolved
	cmd := exec.Command("systemctl", "stop", "systemd-resolved")
	_, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger.Debug(fmt.Sprintf("Failed to stop systemd-resolved: %v", err))
	}
	// stop AP
	accesspoint.Stop()
	// start AP
	if err := accesspoint.Start(dev[0]); err != nil {
		return err
	}
	return nil
}

func C2CConnect(ssid, password string) {
	logger.Logger.Debug("C2C Attempting to connect to ssid")
	UnaliveC2C()
	dev, _ := getWifiDevice()
	runCommand("nmcli", "radio", "wifi", "on")
	time.Sleep(5 * time.Second)
	runCommand("sudo", "ip", "link", "set", dev[0], "up")
	// attempt to connect
	err := ConnectToWifi(ssid, password)
	if err != nil {
		C2CMode()
	} else {
		cmd := exec.Command("systemctl", "restart", "groundseg")
		_, err := cmd.CombinedOutput()
		if err != nil {
			logger.Logger.Debug(fmt.Sprintf("Failed to restart groundseg: %v", err))
		}
	}
}

func UnaliveC2C() error {
	// stop AP
	accesspoint.Stop()
	// start systemd-resolved
	cmd := exec.Command("systemctl", "start", "systemd-resolved")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start systemd-resolved: %v", err)
	}
	return nil
}

func CaptivePortal(dev string) error {
	if err := proxy.Run(); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error creating captive portal: %v", err))
		os.Exit(1)
	}
	return nil
}

func CaptiveAPI(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	clients[conn] = true
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
				logger.Logger.Info("WS closed")
				conn.Close()
			}
			logger.Logger.Warn(fmt.Sprintf("Error reading websocket message: %v", err))
			break
		}
		var payload structs.WsC2cPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
			continue
		}
		if payload.Payload.Action == "connect" {
			if err := ConnectToWifi(payload.Payload.SSID, payload.Payload.Password); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to connect: %v", err))
			} else {
				if _, err := runCommand("systemclt", "restart", "groundseg"); err != nil {
					logger.Logger.Error(fmt.Sprintf("Couldn't restart GroundSeg after connection!"))
				}
			}
		}
	}
}

func announceNetworks(dev string) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			networks := ListWifiSSIDs(dev)
			payload := struct {
				Networks []string `json:"networks"`
			}{
				Networks: networks,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error marshaling payload: %v", err))
				continue
			}
			for client, _ := range clients {
				if client != nil && clients[client] == true {
					if err := client.WriteMessage(websocket.TextMessage, payloadJSON); err != nil {
						logger.Logger.Error(fmt.Sprintf("Error sending message: %v", err))
						clients[client] = false
						continue
					}
				}
			}
		}
	}
}

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func getWifiDevice() ([]string, error) {
	cmd := "nmcli device status | grep wifi | awk '{print $1}'"
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return []string{}, fmt.Errorf("no WiFi device found")
	}
	wifiDevices := strings.Split(strings.TrimSpace(string(out)), "\n")
	if wifiDevices != nil {
		return wifiDevices, nil
	}
	return []string{}, fmt.Errorf("no WiFi device found")
}

func ListWifiSSIDs(dev string) []string {
	out, err := runCommand("nmcli", "-t", "dev", "wifi", "list", "ifname", dev)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't gather wifi networks: %v", err))
		return []string{}
	}
	lines := strings.Split(out, "\n")
	var ssids []string
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) > 6 && parts[7] != "" {
			ssids = append(ssids, parts[7])
		}
	}
	return ssids
}

func getConnectedSSID(c *wifi.Client, dev string) string {
	interfaces, err := c.Interfaces()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't get devices: %v", err))
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
	cmd := exec.Command("nmcli", "dev", "wifi", "connect", ssid, "password", password)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to wifi: %v", err)
	}
	return nil
}

func DisconnectWifi(ifaceName string) error {
	c, err := wifi.New()
	if err != nil {
		return err
	}
	defer c.Close()
	iface := &wifi.Interface{Name: ifaceName}
	return c.Disconnect(iface)
}

func ToggleDevice(dev string) error {
	var cmd string
	if ifCheck() {
		cmd = "off"
	} else {
		cmd = "on"
	}
	_, err := runCommand("nmcli", "radio", "wifi", cmd)
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
		if _, err := runCommand("sysctl", "-w", fmt.Sprintf("%s=%s", key, value)); err != nil {
			return fmt.Errorf("failed to set %s: %v", key, err)
		}
	}
	return nil
}
