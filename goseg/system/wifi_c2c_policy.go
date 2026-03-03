package system

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type C2CModeOrchestrator interface {
	EnterC2CMode() error
	ConnectToNetwork(ssid, password string) error
	ExitC2CMode() error
}

type c2cModeServiceDependencies struct {
	radio          wifiRadioService
	accessPoint    accessPointLifecycle
	getStoredSSIDs func([]string)
}

type c2cModeLifecycleDependencies struct {
	startResolved   func() error
	stopResolved    func() error
	rebootSystem    func() error
	pause           func(time.Duration)
	publishInterval func(string)
}

type c2cModeDependencies struct {
	c2cModeServiceDependencies
	c2cModeLifecycleDependencies
}

type c2cModeFlow struct {
	deps c2cModeDependencies
}

var (
	defaultAccessPointLifecycle accessPointLifecycle = systemAccessPointLifecycle{}
	c2cStartDelay                                    = 10 * time.Second
	c2cConnectDelay                                  = 5 * time.Second
	c2cRebootDelay                                   = 1 * time.Second
)

func defaultC2CModeDependencies() c2cModeDependencies {
	return c2cModeDependencies{
		c2cModeServiceDependencies: c2cModeServiceDependencies{
			radio:          newWiFiRadioService(DefaultWiFiRuntime()),
			accessPoint:    defaultAccessPointLifecycle,
			getStoredSSIDs: func(ssids []string) { C2CStoredSSIDs = ssids },
		},
		c2cModeLifecycleDependencies: c2cModeLifecycleDependencies{
			startResolved:   func() error { return runSystemdResolvedForRuntime(DefaultWiFiRuntime(), "start") },
			stopResolved:    func() error { return runSystemdResolvedForRuntime(DefaultWiFiRuntime(), "stop") },
			rebootSystem:    func() error { return runRebootCommandForRuntime(DefaultWiFiRuntime()) },
			pause:           func(d time.Duration) { time.Sleep(d) },
			publishInterval: func(event string) { ConfChannel() <- event },
		},
	}
}

func sanitizeC2CModeDependencies(overrides c2cModeDependencies) c2cModeDependencies {
	dependencies := defaultC2CModeDependencies()
	serviceOverrides := overrides.c2cModeServiceDependencies
	if serviceOverrides.radio != nil {
		dependencies.radio = serviceOverrides.radio
	}
	if serviceOverrides.accessPoint != nil {
		dependencies.accessPoint = serviceOverrides.accessPoint
	}
	if serviceOverrides.getStoredSSIDs != nil {
		dependencies.getStoredSSIDs = serviceOverrides.getStoredSSIDs
	}
	lifecycleOverrides := overrides.c2cModeLifecycleDependencies
	if lifecycleOverrides.startResolved != nil {
		dependencies.startResolved = lifecycleOverrides.startResolved
	}
	if lifecycleOverrides.stopResolved != nil {
		dependencies.stopResolved = lifecycleOverrides.stopResolved
	}
	if lifecycleOverrides.rebootSystem != nil {
		dependencies.rebootSystem = lifecycleOverrides.rebootSystem
	}
	if lifecycleOverrides.pause != nil {
		dependencies.pause = lifecycleOverrides.pause
	}
	if lifecycleOverrides.publishInterval != nil {
		dependencies.publishInterval = lifecycleOverrides.publishInterval
	}
	return dependencies
}

func NewC2CModeFlow() C2CModeOrchestrator {
	return newC2CModeFlow()
}

func newC2CModeFlow() C2CModeOrchestrator {
	return c2cModeFlow{
		deps: defaultC2CModeDependencies(),
	}
}

func newC2CModeFlowWithDependencies(dependencies c2cModeDependencies) C2CModeOrchestrator {
	return c2cModeFlow{
		deps: sanitizeC2CModeDependencies(dependencies),
	}
}

func (flow c2cModeFlow) EnterC2CMode() error {
	zap.L().Info("C2C Mode initializing")
	if err := flow.deps.radio.Enable(); err != nil {
		return fmt.Errorf("couldn't enable wifi interface: %w", err)
	}

	flow.deps.pause(c2cStartDelay)

	device, err := flow.deps.radio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("failed to discover wifi device for C2C mode: %w", err)
	}

	ssids, err := flow.deps.radio.ListSSIDs(device)
	if err != nil {
		return fmt.Errorf("couldn't list ssids for %s: %w", device, err)
	}
	flow.deps.getStoredSSIDs(ssids)
	zap.L().Info(fmt.Sprintf("C2C retrieved available SSIDs: %v", C2CStoredSSIDs))

	if err := flow.deps.stopResolved(); err != nil {
		return err
	}

	if err := flow.deps.accessPoint.Start(device); err != nil {
		return fmt.Errorf("failed to start access point: %w", err)
	}
	return nil
}

func (flow c2cModeFlow) ConnectToNetwork(ssid, password string) error {
	zap.L().Debug("C2C Attempting to connect to ssid")
	if err := flow.ExitC2CMode(); err != nil {
		return fmt.Errorf("disable C2C access point before wifi connect: %w", err)
	}

	device, err := flow.deps.radio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("discover wifi device for connect: %w", err)
	}
	if err := flow.deps.radio.Enable(); err != nil {
		return fmt.Errorf("start wifi device %s: %w", device, err)
	}

	flow.deps.pause(c2cConnectDelay)

	if err := flow.deps.radio.SetLinkUp(device); err != nil {
		return fmt.Errorf("set wifi interface %s up: %w", device, err)
	}

	err = flow.deps.radio.Connect(ssid, password)
	if err != nil {
		connectErr := fmt.Errorf("connect to wifi %s: %w", ssid, err)
		if c2cErr := flow.EnterC2CMode(); c2cErr != nil {
			return fmt.Errorf("restore C2C mode after failed connect: %w", errors.Join(connectErr, c2cErr))
		}
		return connectErr
	}

	if flow.deps.publishInterval != nil {
		flow.deps.publishInterval("c2cInterval")
	}
	flow.deps.pause(c2cRebootDelay)

	if err := flow.deps.rebootSystem(); err != nil {
		return fmt.Errorf("reboot after C2C connect: %w", err)
	}
	return nil
}

func (flow c2cModeFlow) ExitC2CMode() error {
	device, err := flow.deps.radio.PrimaryDevice()
	if err != nil {
		return fmt.Errorf("failed to discover wifi device for C2C shutdown: %w", err)
	}
	if err := flow.deps.accessPoint.Stop(device); err != nil {
		return fmt.Errorf("failed to stop access point on %s: %w", device, err)
	}
	return flow.deps.startResolved()
}

func runSystemdResolvedForRuntime(runtime wifiRuntime, mode string) error {
	_, err := runtime.RunCommand("systemctl", mode, "systemd-resolved")
	if err != nil {
		return fmt.Errorf("failed to %s systemd-resolved: %w", mode, err)
	}
	return nil
}

// EnableResolved starts the systemd-resolved service for default runtime operations.
func EnableResolved() error {
	_, err := DefaultWiFiRuntime().RunCommand("systemctl", "start", "systemd-resolved")
	if err != nil {
		return fmt.Errorf("failed to start systemd-resolved: %w", err)
	}
	return nil
}

func runRebootCommandForRuntime(runtime wifiRuntime) error {
	cmd := runtime.ExecCommand("reboot")
	_, err := cmd.CombinedOutput()
	return err
}
