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
	return NewWiFiRuntime()
}

func NewWiFiRuntimeService(runtime ...wifiRuntime) *WiFiRuntimeService {
	runtimeInstance := NewWiFiRuntime()
	if len(runtime) > 0 {
		runtimeInstance = NewWiFiRuntimeWith(runtime[0])
	}
	return &WiFiRuntimeService{
		runtime: runtimeInstance,
		state:   DefaultWiFiRuntimeState(),
	}
}

func (service *WiFiRuntimeService) prepareForUse() (wifiRuntime, *wifiRuntimeState, error) {
	if service == nil {
		return wifiRuntime{}, nil, fmt.Errorf("wifi runtime service is nil")
	}

	if service.runtime.RunCommand == nil {
		service.runtime = NewWiFiRuntime()
	}
	if service.runtime.RunCommand == nil {
		return wifiRuntime{}, nil, fmt.Errorf("wifi runtime failed to initialize")
	}

	if service.state == nil {
		service.state = DefaultWiFiRuntimeState()
	}
	if service.state == nil {
		return wifiRuntime{}, nil, fmt.Errorf("wifi runtime state failed to initialize")
	}

	return service.runtime, service.state, nil
}

func (service *WiFiRuntimeService) IsWifiEnabled() (bool, error) {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return false, err
	}
	return runtime.ifCheck()
}

func (service *WiFiRuntimeService) ConstructWifiInfo(dev string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return err
	}

	device := strings.TrimSpace(dev)
	if device == "" {
		device = WiFiDevice(state)
	}
	if device == "" {
		return fmt.Errorf("wifi device could not be resolved")
	}

	newWiFiRadioService(runtime, state).RefreshInfo(device)
	return nil
}

func (service *WiFiRuntimeService) ListWifiSSIDs(dev string) ([]string, error) {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return nil, err
	}

	device, err := resolveDevice(dev, runtime, state)
	if err != nil {
		return nil, fmt.Errorf("resolve wifi device: %w", err)
	}
	return runtime.listSSIDs(device)
}

func (service *WiFiRuntimeService) ConnectToWifi(ssid, password string) error {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return err
	}

	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("ssid cannot be empty")
	}

	enabled, err := runtime.ifCheck()
	if err != nil {
		return fmt.Errorf("check wifi radio before connect: %w", err)
	}
	if !enabled {
		if err := runtime.toggleDevice(""); err != nil {
			return fmt.Errorf("enable wifi radio before connect: %w", err)
		}
	}
	return runtime.connect(ssid, password)
}

func (service *WiFiRuntimeService) DisconnectWifi(ifaceName string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return err
	}

	device, err := resolveDevice(ifaceName, runtime, state)
	if err != nil {
		return fmt.Errorf("resolve wifi device for disconnect: %w", err)
	}
	return runtime.disconnect(device)
}

func (service *WiFiRuntimeService) ToggleDevice(dev string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return err
	}

	device, err := resolveDevice(dev, runtime, state)
	if err != nil {
		return fmt.Errorf("resolve wifi device for toggle: %w", err)
	}
	return runtime.toggleDevice(device)
}

func (service *WiFiRuntimeService) StartWiFiInfoLoop(ctx context.Context) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return err
	}
	return startWiFiInfoLoop(ctx, runtime, state)
}

func (service *WiFiRuntimeService) StopWiFiInfoLoop() {
	_, state, err := service.prepareForUse()
	if err != nil {
		return
	}

	state.stopMu.Lock()
	stop := state.wifiInfoLoopStop
	state.wifiInfoLoopStop = nil
	state.stopMu.Unlock()

	if stop != nil {
		stop()
	}
}

func (service *WiFiRuntimeService) ApplyCaptiveRules() error {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return err
	}
	return runtime.applyCaptiveRules()
}

func resolveDevice(dev string, runtime wifiRuntime, state *wifiRuntimeState) (string, error) {
	device := strings.TrimSpace(dev)
	if device != "" {
		return device, nil
	}
	if WiFiDevice(state) != "" {
		return WiFiDevice(state), nil
	}
	return runtime.primaryWifiDevice()
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
