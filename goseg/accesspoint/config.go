package accesspoint

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

var (
	hostapdInterfacePattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)
	hostapdSSIDPattern      = regexp.MustCompile(`^[a-zA-Z0-9 _.-]{1,32}$`)
	hostapdPassPattern      = regexp.MustCompile(`^[\x21-\x7E]{8,63}$`)
)

func writeHostapdConfig(configPath, wlanInterface, networkSSID, passphrase string) error {
	config, err := buildHostapdConfig(wlanInterface, networkSSID, passphrase)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("Hostapd config saved to %s", configPath))
	return nil
}

func buildHostapdConfig(wlan, ssid, wpaPassphrase string) (string, error) {
	if !hostapdInterfacePattern.MatchString(wlan) {
		return "", fmt.Errorf("invalid wlan interface %q", wlan)
	}
	if !hostapdSSIDPattern.MatchString(ssid) {
		return "", fmt.Errorf("invalid ssid %q", ssid)
	}
	if !hostapdPassPattern.MatchString(wpaPassphrase) {
		return "", fmt.Errorf("invalid WPA passphrase format")
	}

	sections := []string{
		buildInterfaceSection(wlan, ssid),
		buildModeSection(),
		buildWPASection(wpaPassphrase),
		buildWEPSection(),
	}
	return strings.Join(sections, "\n"), nil
}

func buildInterfaceSection(wlan, ssid string) string {
	var lines = []string{
		"#sets the wifi interface to use, is wlan0 in most cases",
		fmt.Sprintf("interface=%s", wlan),
		"#driver to use, nl80211 works in most cases",
		"driver=nl80211",
		"#sets the ssid of the virtual wifi access point",
		fmt.Sprintf("ssid=%s", ssid),
	}
	return strings.Join(lines, "\n")
}

func buildModeSection() string {
	var lines = []string{
		"#sets the mode of wifi, depends upon the devices you will be using. It can be a,b,g,n. Setting to g ensures backward compatiblity.",
		"hw_mode=g",
		"#sets the channel for your wifi",
		"channel=6",
		"#macaddr_acl sets options for mac address filtering. 0 means \"accept unless in deny list\"",
		"macaddr_acl=0",
		"#setting ignore_broadcast_ssid to 1 will disable the broadcasting of ssid",
		"ignore_broadcast_ssid=0",
		"#Sets authentication algorithm",
		"#1 - only open system authentication",
		"#2 - both open system authentication and shared key authentication",
		"auth_algs=1",
	}
	return strings.Join(lines, "\n")
}

func buildWPASection(wpaPassphrase string) string {
	var lines = []string{
		"#####Sets WPA and WPA2 authentication#####",
		"#wpa option sets which wpa implementation to use",
		"#1 - wpa only",
		"#2 - wpa2 only",
		"#3 - both",
		"wpa=3",
		"#sets wpa passphrase required by the clients to authenticate themselves on the network",
		fmt.Sprintf("wpa_passphrase=%s", wpaPassphrase),
		"#sets wpa key management",
		"wpa_key_mgmt=WPA-PSK",
		"#sets encryption used by WPA",
		"wpa_pairwise=TKIP",
		"#sets encryption used by WPA2",
		"rsn_pairwise=CCMP",
	}
	return strings.Join(lines, "\n")
}

func buildWEPSection() string {
	return strings.Join([]string{
		"#################################",
		"#####Sets WEP authentication#####",
		"#WEP is not recommended as it can be easily broken into",
		"#wep_default_key=0",
		"#wep_key0=qwert    #5,13, or 16 characters",
		"#optionally you may also define wep_key2, wep_key3, and wep_key4",
		"#################################",
		"#For No encryption, you do not need to set any options",
	}, "\n")
}
