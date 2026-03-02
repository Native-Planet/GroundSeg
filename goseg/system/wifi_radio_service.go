package system

import (
	"fmt"

	"go.uber.org/zap"
	"groundseg/structs"
)

type wifiRadioService interface {
	PrimaryDevice() (string, error)
	RefreshInfo(device string)
	Enable() error
	SetLinkUp(device string) error
	Connect(ssid, password string) error
	ListSSIDs(device string) ([]string, error)
}

type nmcliWiFiRadioService struct{}

func (nmcliWiFiRadioService) PrimaryDevice() (string, error) {
	return defaultWiFiService.primaryWifiDevice()
}

func (nmcliWiFiRadioService) RefreshInfo(device string) {
	info := structs.SystemWifi{Status: false}
	wifiEnabled, err := defaultWiFiService.ifCheck()
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't read wifi radio state: %v", err))
		setWifiInfo(info)
		return
	}

	info.Status = wifiEnabled
	if !info.Status {
		setWifiInfo(info)
		return
	}

	client, err := defaultWiFiService.runtime.newWifiClient()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't create wifi client with device %v: %v", device, err))
		info.Status = false
		setWifiInfo(info)
		return
	}
	defer client.Close()

	active, err := defaultWiFiService.connectedSSID(client, device)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get active SSID for %s: %v", device, err))
		info.Active = ""
	} else {
		info.Active = active
	}

	ssids, err := defaultWiFiService.listSSIDs(device)
	if err != nil {
		zap.L().Error(err.Error())
		info.Networks = []string{}
	} else {
		info.Networks = ssids
	}

	setWifiInfo(info)
}

func (nmcliWiFiRadioService) Enable() error {
	if _, err := defaultWiFiService.runtime.runCommand("nmcli", "radio", "wifi", "on"); err != nil {
		return fmt.Errorf("enable wifi radio: %w", err)
	}
	return nil
}

func (nmcliWiFiRadioService) SetLinkUp(device string) error {
	if _, err := defaultWiFiService.runtime.runCommand("sudo", "ip", "link", "set", device, "up"); err != nil {
		return fmt.Errorf("set ip link for device %s: %w", device, err)
	}
	return nil
}

func (nmcliWiFiRadioService) ListSSIDs(device string) ([]string, error) {
	return defaultWiFiService.listSSIDs(device)
}

func (nmcliWiFiRadioService) Connect(ssid, password string) error {
	return defaultWiFiService.connect(ssid, password)
}
