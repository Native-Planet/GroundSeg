package system

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/mdlayher/wifi"
	"github.com/schollz/wifiscan"
)

func C2cMode() error {
	dev, err := getWifiDevice()
	if err != nil {
		return err
	}
	if err := setupHostAPD(dev); err != nil {
		return err
	}
	return nil
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

func listWifiSSIDs() ([]string, error) {
	var ssids []string
	wifis, err := wifiscan.Scan()
	if err != nil {
		return nil, err
	}
	for _, w := range wifis {
		ssids = append(ssids, w.SSID)
	}
	return ssids, nil
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

func teardownHostAPD() error {
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
