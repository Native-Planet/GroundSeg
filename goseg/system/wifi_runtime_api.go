package system

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrWifiInterfaceNotFound = errors.New("wifi interface not found")

type WiFiRuntimeService struct {
	runtime wifiRuntime
	state   *wifiRuntimeState
}

func resolveWiFiRuntime(overrides ...wifiRuntime) wifiRuntime {
	if len(overrides) > 0 {
		return NewWiFiRuntimeWith(overrides[0])
	}
	return DefaultWiFiRuntime()
}

func NewWiFiRuntimeService(runtime ...wifiRuntime) *WiFiRuntimeService {
	runtimeInstance := DefaultWiFiRuntime()
	if len(runtime) > 0 {
		runtimeInstance = NewWiFiRuntimeWith(runtime[0])
	}
	return &WiFiRuntimeService{
		runtime: runtimeInstance,
		state:   DefaultWiFiRuntimeState(),
	}
}

func DefaultWiFiRuntimeService() *WiFiRuntimeService {
	return NewWiFiRuntimeService()
}

func (service *WiFiRuntimeService) IsWifiEnabled() (bool, error) {
	if service == nil {
		return false, errors.New("wifi runtime service is nil")
	}
	return service.runtime.ifCheck()
}

func (service *WiFiRuntimeService) ConstructWifiInfo(dev string) {
	device := strings.TrimSpace(dev)
	if device == "" {
		device = WiFiDevice(service.state)
	}
	if device == "" || service == nil {
		return
	}
	newWiFiRadioService(service.runtime, service.state).RefreshInfo(device)
}

func (service *WiFiRuntimeService) ListWifiSSIDs(dev string) ([]string, error) {
	device, err := service.resolveDevice(dev)
	if err != nil {
		return nil, fmt.Errorf("resolve wifi device: %w", err)
	}
	return service.runtime.listSSIDs(device)
}

func (service *WiFiRuntimeService) ConnectToWifi(ssid, password string) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("ssid cannot be empty")
	}
	enabled, err := service.runtime.ifCheck()
	if err != nil {
		return fmt.Errorf("check wifi radio before connect: %w", err)
	}
	if !enabled {
		if err := service.runtime.toggleDevice(""); err != nil {
			return fmt.Errorf("enable wifi radio before connect: %w", err)
		}
	}
	return service.runtime.connect(ssid, password)
}

func (service *WiFiRuntimeService) DisconnectWifi(ifaceName string) error {
	device := strings.TrimSpace(ifaceName)
	if device == "" {
		var err error
		device, err = service.runtime.primaryWifiDevice()
		if err != nil {
			return fmt.Errorf("resolve wifi device for disconnect: %w", err)
		}
	}
	return service.runtime.disconnect(device)
}

func (service *WiFiRuntimeService) ToggleDevice(dev string) error {
	if service == nil {
		return fmt.Errorf("wifi runtime service is nil")
	}
	target := strings.TrimSpace(dev)
	_, err := service.runtime.primaryWifiDevice()
	if err != nil {
		return fmt.Errorf("resolve wifi device for toggle: %w", err)
	}
	if target == "" {
		target = WiFiDevice(service.state)
	}
	return service.runtime.toggleDevice(target)
}

func (service *WiFiRuntimeService) StartWiFiInfoLoop(ctx context.Context) error {
	return startWiFiInfoLoop(ctx, service.runtime, service.state)
}

func (service *WiFiRuntimeService) StopWiFiInfoLoop() {
	state := service.state
	state.stopMu.Lock()
	stop := state.wifiInfoLoopStop
	state.wifiInfoLoopStop = nil
	state.stopMu.Unlock()
	if stop != nil {
		stop()
	}
}

func (service *WiFiRuntimeService) ApplyCaptiveRules() error {
	if service == nil {
		return fmt.Errorf("wifi runtime service is nil")
	}
	return service.runtime.applyCaptiveRules()
}

func (service *WiFiRuntimeService) resolveDevice(dev string) (string, error) {
	device := strings.TrimSpace(dev)
	if device != "" {
		return device, nil
	}
	if service == nil {
		return "", fmt.Errorf("wifi runtime service is nil")
	}
	if WiFiDevice(service.state) != "" {
		return WiFiDevice(service.state), nil
	}
	return service.runtime.primaryWifiDevice()
}

func connectInfoLoop(ctx context.Context, runtime wifiRuntime, radio wifiRadioService, state *wifiRuntimeState) error {
	device, err := radio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("couldn't find a wifi device: %w", err)
	}
	{
		setWiFiDevice(device, state)
	}
	radio.RefreshInfo(device)

	state.stopMu.Lock()
	if state.wifiInfoLoopStop != nil {
		state.stopMu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	state.wifiInfoLoopStop = cancel
	state.stopMu.Unlock()
	go wifiInfoLoop(ctx, device, runtime, radio)
	return nil
}

func startWiFiInfoLoop(ctx context.Context, runtime wifiRuntime, state *wifiRuntimeState) error {
	return startWiFiInfoLoopWithRadio(ctx, runtime, newWiFiRadioService(runtime, state), state)
}

func startWiFiInfoLoopWithRadio(ctx context.Context, runtime wifiRuntime, radio wifiRadioService, state *wifiRuntimeState) error {
	return connectInfoLoop(ctx, runtime, radio, state)
}

func wifiInfoLoop(ctx context.Context, dev string, runtime wifiRuntime, radio wifiRadioService) {
	ticker := runtime.WifiInfoTicker()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			radio.RefreshInfo(dev)
		}
	}
}
