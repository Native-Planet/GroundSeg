package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	C2cChan = make(chan bool)
	proxy   = &captive.Portal{
		LoginPath:           "/",
		PortalDomain:        "nativeplanet.local",
		AllowedBypassPortal: false,
		WebPath:             "c2c",
	}
	WifiInfo structs.SystemWifi
	Device   string
)

func init() {
	dev, err := getWifiDevice()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't find a wifi device! %v", err))
	} else {
		Device = dev
		constructWifiInfo(dev)
		go wifiInfoLoop(dev)
	}
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
			logger.Logger.Error(fmt.Sprintf("Couldn't create wifi client: %v", err))
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

func C2cMode() error {
	dev, err := getWifiDevice()
	if err != nil {
		return err
	}
	if err := setupHostAPD(dev); err != nil {
		return err
	}
	go announceNetworks(dev)
	if err := CaptivePortal(dev); err != nil {
		return err
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
	dev, _ := getWifiDevice()
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
			if err := ConnectToWifi(dev, payload.Payload.SSID, payload.Payload.Password); err != nil {
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

func getWifiDevice() (string, error) {
	cmd := "nmcli device status | grep wifi | grep -v p2p | awk '{print $1}'"
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("no WiFi device found")
	}
	wifiDevice := strings.TrimSpace(string(out))
	if wifiDevice != "" {
		return wifiDevice, nil
	}
	return "", fmt.Errorf("no WiFi device found")
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

func ConnectToWifi(ifaceName, ssid, password string) error {
	c, err := wifi.New()
	if err != nil {
		return err
	}
	defer c.Close()
	iface := &wifi.Interface{Name: ifaceName}
	if password == "" {
		return c.Connect(iface, ssid)
	}
	return c.ConnectWPAPSK(iface, ssid, password)
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

func setupHostAPD(iface string) error {
	hostapdConf := `
interface=` + iface + `
driver=nl80211
ssid=nativeplanet
hw_mode=g
channel=6
macaddr_acl=0
auth_algs=1
ignore_broadcast_ssid=0
wpa=2
wpa_passphrase=nativeplanet
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP
rsn_pairwise=CCMP
`
	err := ioutil.WriteFile("/etc/hostapd/hostapd.conf", []byte(hostapdConf), 0644)
	if err != nil {
		return err
	}
	_, err = runCommand("hostapd", "/etc/hostapd/hostapd.conf")
	if err != nil {
		return err
	}
	_, err = runCommand("ifconfig", iface, "10.0.0.1", "netmask", "255.255.255.0", "up")
	if err != nil {
		return err
	}
	_, err = runCommand("dnsmasq", "--interface="+iface, "--bind-interfaces", "--dhcp-range=10.0.0.2,10.0.0.20,12h")
	if err != nil {
		return err
	}
	return nil
}

func TeardownHostAPD() error {
	_, err := runCommand("pkill", "-9", "hostapd")
	if err != nil {
		return err
	}
	_, err = runCommand("pkill", "-9", "dnsmasq")
	if err != nil {
		return err
	}
	if err := proxy.Close(); err != nil {
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
