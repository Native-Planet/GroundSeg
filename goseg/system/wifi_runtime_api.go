package system

import (
	"context"
	"errors"
	"fmt"
	"groundseg/internal/seams"
	"os/exec"
	"strings"
)

var ErrWiFiInterfaceNotFound = errors.New("wifi interface not found")

type WiFiRuntimeService struct {
	runtime         wifiRuntime
	state           *wifiRuntimeState
	prepareErr      error
	runtimePrepared bool
}

func resolveWiFiRuntime(overrides ...wifiRuntime) wifiRuntime {
	if len(overrides) > 0 {
		return NewWiFiRuntimeWith(overrides[0])
	}
	return NewWiFiRuntime()
}

func NewWiFiRuntimeService(runtime ...wifiRuntime) *WiFiRuntimeService {
	runtimeInstance := resolveWiFiRuntime(runtime...)
	service := &WiFiRuntimeService{
		runtime: runtimeInstance,
		state:   DefaultWiFiRuntimeState(),
	}
	if err := validateWiFiRuntime(service.runtime); err != nil {
		service.prepareErr = err
	} else {
		service.runtimePrepared = true
	}
	return service
}

func validateWiFiRuntime(runtime wifiRuntime) error {
	if runtime.ExecCommand == nil {
		return seams.MissingRuntimeDependency("wifi runtime ExecCommand callback", "")
	}
	if runtime.RunCommand == nil {
		return seams.MissingRuntimeDependency("wifi runtime RunCommand callback", "")
	}
	if runtime.RunNmcliFn == nil {
		return seams.MissingRuntimeDependency("wifi runtime RunNmcliFn callback", "")
	}
	if runtime.NewWifiClient == nil {
		return seams.MissingRuntimeDependency("wifi runtime NewWifiClient callback", "")
	}
	if runtime.ClientInterfacesFn == nil {
		return seams.MissingRuntimeDependency("wifi runtime ClientInterfacesFn callback", "")
	}
	if runtime.ClientBSSFn == nil {
		return seams.MissingRuntimeDependency("wifi runtime ClientBSSFn callback", "")
	}
	if runtime.WifiInfoTicker == nil {
		return seams.MissingRuntimeDependency("wifi runtime WifiInfoTicker callback", "")
	}
	return nil
}

func (service *WiFiRuntimeService) prepareForUse() (wifiRuntime, *wifiRuntimeState, error) {
	if service == nil {
		return wifiRuntime{}, nil, fmt.Errorf("wifi runtime service is nil")
	}
	if !service.runtimePrepared {
		service.runtime = resolveWiFiRuntime(service.runtime)
		if err := validateWiFiRuntime(service.runtime); err != nil {
			service.prepareErr = err
		}
		service.runtimePrepared = true
	}
	if service.prepareErr != nil {
		return wifiRuntime{}, nil, service.prepareErr
	}

	if service.state == nil {
		service.state = DefaultWiFiRuntimeState()
	}
	if service.state == nil {
		return wifiRuntime{}, nil, fmt.Errorf("wifi runtime state failed to initialize")
	}

	return service.runtime, service.state, nil
}

func (service *WiFiRuntimeService) IsWiFiEnabled() (bool, error) {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return false, fmt.Errorf("prepare wifi runtime for IsWiFiEnabled: %w", err)
	}
	enabled, err := runtime.isWiFiRadioEnabled()
	if err != nil {
		return false, fmt.Errorf("read wifi enabled state: %w", err)
	}
	return enabled, nil
}

func (service *WiFiRuntimeService) RefreshWiFiInfo(interfaceName string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for RefreshWiFiInfo: %w", err)
	}

	device := strings.TrimSpace(interfaceName)
	if device == "" {
		device = WiFiDevice(state)
	}
	if device == "" {
		return fmt.Errorf("wifi device could not be resolved")
	}

	newWiFiRadioService(runtime, state).RefreshInfo(device)
	return nil
}

func (service *WiFiRuntimeService) ListWiFiSSIDs(interfaceName string) ([]string, error) {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return nil, fmt.Errorf("prepare wifi runtime for ListWiFiSSIDs: %w", err)
	}

	device, err := resolveInterfaceName(interfaceName, runtime, state)
	if err != nil {
		return nil, fmt.Errorf("resolve wifi device: %w", err)
	}
	ssids, err := runtime.listSSIDs(device)
	if err != nil {
		return nil, fmt.Errorf("list wifi SSIDs for %s: %w", device, err)
	}
	return ssids, nil
}

func (service *WiFiRuntimeService) ConnectToWiFi(ssid, password string) error {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for ConnectToWiFi: %w", err)
	}

	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("ssid cannot be empty")
	}

	enabled, err := runtime.isWiFiRadioEnabled()
	if err != nil {
		return fmt.Errorf("check wifi radio before connect: %w", err)
	}
	if !enabled {
		if err := runtime.toggleDevice(""); err != nil {
			return fmt.Errorf("enable wifi radio before connect: %w", err)
		}
	}
	if err := runtime.connect(ssid, password); err != nil {
		return fmt.Errorf("connect to wifi SSID %q: %w", ssid, err)
	}
	return nil
}

func (service *WiFiRuntimeService) DisconnectWiFi(interfaceName string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for DisconnectWiFi: %w", err)
	}

	device, err := resolveInterfaceName(interfaceName, runtime, state)
	if err != nil {
		return fmt.Errorf("resolve wifi device for disconnect: %w", err)
	}
	if err := runtime.disconnect(device); err != nil {
		return fmt.Errorf("disconnect wifi interface %s: %w", device, err)
	}
	return nil
}

func (service *WiFiRuntimeService) ToggleDevice(interfaceName string) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for ToggleDevice: %w", err)
	}

	device, err := resolveInterfaceName(interfaceName, runtime, state)
	if err != nil {
		return fmt.Errorf("resolve wifi device for toggle: %w", err)
	}
	if err := runtime.toggleDevice(device); err != nil {
		return fmt.Errorf("toggle wifi device %s: %w", device, err)
	}
	return nil
}

func (service *WiFiRuntimeService) StartWiFiInfoLoop(ctx context.Context) error {
	runtime, state, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for StartWiFiInfoLoop: %w", err)
	}
	if err := startWiFiInfoLoop(ctx, runtime, state); err != nil {
		return fmt.Errorf("start wifi info loop: %w", err)
	}
	return nil
}

func (service *WiFiRuntimeService) StopWiFiInfoLoop() error {
	_, state, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for StopWiFiInfoLoop: %w", err)
	}

	state.stopMu.Lock()
	stop := state.wifiInfoLoopStop
	state.wifiInfoLoopStop = nil
	state.stopMu.Unlock()

	if stop != nil {
		stop()
	}
	return nil
}

func (service *WiFiRuntimeService) ApplyCaptiveRules() error {
	runtime, _, err := service.prepareForUse()
	if err != nil {
		return fmt.Errorf("prepare wifi runtime for ApplyCaptiveRules: %w", err)
	}
	if err := runtime.applyCaptiveRules(); err != nil {
		return fmt.Errorf("apply captive rules: %w", err)
	}
	return nil
}

func resolveInterfaceName(interfaceName string, runtime wifiRuntime, state *wifiRuntimeState) (string, error) {
	device := strings.TrimSpace(interfaceName)
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

func wifiInfoLoop(ctx context.Context, interfaceName string, runtime wifiRuntime, radio wifiRadioService) {
	ticker := runtime.WifiInfoTicker()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			radio.RefreshInfo(interfaceName)
		}
	}
}

var rebootHostCommand = func() error {
	cmd := exec.Command("reboot")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reboot host: %w", err)
	}
	return nil
}

// RebootHost performs a host reboot using the system runtime command path.
// It is factored as a seam for safe test overrides.
func RebootHost() error {
	return rebootHostCommand()
}
