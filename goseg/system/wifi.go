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
)

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
	proxy := &captive.Portal{
		LoginPath:           "/",
		PortalDomain:        "nativeplanet.local",
		AllowedBypassPortal: false,
		WebPath:             "c2c",
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.Dir("c2c"))
		})
		http.Handle("/api", http.HandlerFunc(captiveAPI))
		http.ListenAndServe(":80", nil)
	}()
	err := proxy.Run()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}
	return nil
}

func captiveAPI(w http.ResponseWriter, r *http.Request) {
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
			if err := connectToWifi(dev, payload.Payload.SSID, payload.Payload.Password); err != nil {
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
			networks := listWifiSSIDs(dev)
			for client, _ := range clients {
				if client != nil && clients[client] == true {
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
	c, err := wifi.New()
	if err != nil {
		return "", err
	}
	defer c.Close()
	devices, err := c.Interfaces()
	if err != nil {
		return "", err
	}
	for _, device := range devices {
		if device.Type == wifi.InterfaceTypeStation {
			return device.Name, nil
		}
	}
	return "", fmt.Errorf("no WiFi device found")
}

func listWifiSSIDs(output string) []string {
	out, err := runCommand("nmcli", "-t", "dev", "wifi")
	if err != nil {
		return nil
	}
	lines := strings.Split(out, "\n")
	var ssids []string
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) > 2 {
			ssids = append(ssids, parts[2])
		}
	}
	return ssids
}

func connectToWifi(ifaceName, ssid, password string) error {
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

func disconnectWifi(ifaceName string) error {
	c, err := wifi.New()
	if err != nil {
		return err
	}
	defer c.Close()
	iface := &wifi.Interface{Name: ifaceName}
	return c.Disconnect(iface)
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
