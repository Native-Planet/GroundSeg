package system

import (
	"fmt"
	"groundseg/structs"

	"go.uber.org/zap"
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
	return primaryWifiDevice()
}

func (nmcliWiFiRadioService) RefreshInfo(device string) {
	info := structs.SystemWifi{
		Status: ifCheckForWiFi(),
	}
	if !info.Status {
		setWifiInfo(info)
		return
	}
	client, err := wifiNewClientForWiFi()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't create wifi client with device %v: %v", device, err))
		info.Status = false
		setWifiInfo(info)
		return
	}
	defer client.Close()

	info.Active = getConnectedSSID(client, device)
	ssids, err := ListWifiSSIDs(device)
	if err != nil {
		zap.L().Error(err.Error())
		info.Networks = []string{}
	} else {
		info.Networks = ssids
	}
	setWifiInfo(info)
}

func (nmcliWiFiRadioService) Enable() error {
	if _, err := runCommandForWiFi("nmcli", "radio", "wifi", "on"); err != nil {
		return fmt.Errorf("enable wifi radio: %w", err)
	}
	return nil
}

func (nmcliWiFiRadioService) SetLinkUp(device string) error {
	if _, err := runCommandForWiFi("sudo", "ip", "link", "set", device, "up"); err != nil {
		return fmt.Errorf("set ip link for device %s: %w", device, err)
	}
	return nil
}

func (nmcliWiFiRadioService) ListSSIDs(device string) ([]string, error) {
	return ListWifiSSIDs(device)
}

func (nmcliWiFiRadioService) Connect(ssid, password string) error {
	return ConnectToWifi(ssid, password)
}
