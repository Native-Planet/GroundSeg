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

type nmcliWiFiRadioService struct {
	runtime wifiRuntime
	state   *wifiRuntimeState
}

func newWiFiRadioService(overrides wifiRuntime, state ...*wifiRuntimeState) wifiRadioService {
	return nmcliWiFiRadioService{
		runtime: resolveWiFiRuntime(overrides),
		state:   resolveWiFiRuntimeState(state...),
	}
}

func (service nmcliWiFiRadioService) PrimaryDevice() (string, error) {
	return service.runtime.primaryWifiDevice()
}

func (service nmcliWiFiRadioService) RefreshInfo(device string) {
	info := structs.SystemWifi{Status: false}
	wifiEnabled, err := service.runtime.ifCheck()
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't read wifi radio state: %v", err))
		setWifiInfo(info, service.state)
		return
	}

	info.Status = wifiEnabled
	if !info.Status {
		setWifiInfo(info, service.state)
		return
	}

	client, err := service.runtime.NewWifiClient()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't create wifi client with device %v: %v", device, err))
		info.Status = false
		setWifiInfo(info, service.state)
		return
	}
	defer client.Close()

	active, err := service.runtime.connectedSSID(client, device)
	if err != nil {
		zap.L().Error(fmt.Sprintf("couldn't get active SSID for %s: %v", device, err))
		info.Active = ""
	} else {
		info.Active = active
	}

	ssids, err := service.runtime.listSSIDs(device)
	if err != nil {
		zap.L().Error(err.Error())
		info.Networks = []string{}
	} else {
		info.Networks = ssids
	}

	setWifiInfo(info, service.state)
}

func (service nmcliWiFiRadioService) Enable() error {
	if _, err := service.runtime.RunCommand("nmcli", "radio", "wifi", "on"); err != nil {
		return fmt.Errorf("enable wifi radio: %w", err)
	}
	return nil
}

func (service nmcliWiFiRadioService) SetLinkUp(device string) error {
	if _, err := service.runtime.RunCommand("sudo", "ip", "link", "set", device, "up"); err != nil {
		return fmt.Errorf("set ip link for device %s: %w", device, err)
	}
	return nil
}

func (service nmcliWiFiRadioService) ListSSIDs(device string) ([]string, error) {
	return service.runtime.listSSIDs(device)
}

func (service nmcliWiFiRadioService) Connect(ssid, password string) error {
	return service.runtime.connect(ssid, password)
}
