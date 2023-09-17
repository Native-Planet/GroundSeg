package system

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
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
	out, err := runCommand("nmcli", "device")
	if err != nil {
		return "", fmt.Errorf("Couldn't list wifi devices: %v", err)
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		words := strings.Fields(line)
		if len(words) >= 2 && words[1] == "wifi" {
			return words[0], nil
		}
	}
	return "", fmt.Errorf("No wifi device detected")
}

func listWifiSSIDs() []string {
	var ssids []string
	out, err := runCommand("nmcli", "-t", "dev", "wifi", "list")
	if err != nil {
		return ssids
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) > 2 {
			ssids = append(ssids, parts[2])
		}
	}
	return ssids
}

func connectToWifi(ssid, password string) error {
	out, err := runCommand("nmcli", "dev", "wifi", "connect", ssid, "password", "\"network-password\"")
	if err != nil {
		return fmt.Errorf("Couldn't connect to wifi network %v: %v", ssid, err)
	}
	fmt.Println(out)
	return nil
}

func disconnectWifi(ssid string) error {
	out, err := runCommand("nmcli", "con", "down", ssid)
	if err != nil {
		return fmt.Errorf("Error disconnecting from wifi network: %v", err)
	}
	fmt.Println(out)
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
