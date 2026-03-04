package shipworkflow

import (
	"errors"
	"fmt"
	"time"

	"groundseg/config"
	"groundseg/internal/workflow"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

func HandleStartramRestart() error {
	return runStartramRestartWithRuntime(defaultStartramRuntime())
}

func runStartramRestartWithRuntime(runtime startramRuntime) error {
	runtime = resolveStartramRuntime(runtime)
	zap.L().Info("Restarting StarTram")
	settings := runtime.StartramSettingsSnapshotFn()
	steps := buildStartramRestartTransitionSteps(runtime, settings)
	return runStartramTransitionTemplate(
		runtime,
		startramTransitionTemplate{
			transitionType: transition.StartramTransitionRestart,
			startEvent:     startramEvent(transition.StartramTransitionRestart, "startram"),
			successEvent:   startramEvent(transition.StartramTransitionRestart, string(transition.StartramTransitionDone)),
			emitSuccess:    true,
			clearEvent:     startramEvent(transition.StartramTransitionRestart, ""),
			clearDelay:     3 * time.Second,
		},
		steps...,
	)
}

func buildStartramRestartTransitionSteps(runtime startramRuntime, settings config.StartramSettings) []transitionStep[structs.Event] {
	steps := []transitionStep[structs.Event]{
		{
			Run: buildStartramRestartGateStep(settings),
		},
	}
	if !settings.WgOn {
		return steps
	}
	return append(steps,
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionRestart, "urbits"),
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionRestart, "minios"),
		},
		transitionStep[structs.Event]{
			Run: buildStartramRestartFleetRecoveryStep(runtime, settings.Piers),
		},
	)
}

func buildStartramRestartGateStep(settings config.StartramSettings) func() error {
	return func() error {
		if !settings.WgOn {
			return fmt.Errorf("startram is disabled")
		}
		return nil
	}
}

func buildStartramRestartFleetRecoveryStep(runtime startramRuntime, piers []string) func() error {
	return func() error {
		zap.L().Info("Recreating containers")
		return runtime.RecoverWireguardFleetFn(piers, true)
	}
}

func HandleStartramToggle() error {
	return runStartramToggleWithRuntime(defaultStartramRuntime())
}

func runStartramToggleWithRuntime(runtime startramRuntime) error {
	runtime = resolveStartramRuntime(runtime)
	settings := runtime.StartramSettingsSnapshotFn()
	steps, buildErr := buildStartramToggleTransitionSteps(runtime, settings)
	if buildErr != nil {
		return buildErr
	}
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionToggle,
		startEvent:     startramEvent(transition.StartramTransitionToggle, transition.StartramTransitionLoading),
		clearEvent:     startramEvent(transition.StartramTransitionToggle, nil),
		clearDelay:     3 * time.Second,
	},
		steps...)
}

func buildStartramToggleTransitionSteps(runtime startramRuntime, settings config.StartramSettings) ([]transitionStep[structs.Event], error) {
	orchestry, err := buildStartramToggleRuntimeSteps(runtime, settings)
	if err != nil {
		return nil, err
	}
	return []transitionStep[structs.Event]{
		{
			Run: buildTransitionStepsRunner(orchestry, "startram toggle"),
		},
	}, nil
}

func buildTransitionStepsRunner(steps []workflow.Step, runbook string) func() error {
	return func() error {
		return runOrchestrationSteps(steps, runbook)
	}
}

func buildStartramToggleRuntimeSteps(runtime startramRuntime, settings config.StartramSettings) ([]workflow.Step, error) {
	coordinator := startramToggleTransitionCoordinator{
		runtime:  runtime,
		settings: settings,
	}
	steps, err := coordinator.buildWireguardSteps()
	if err != nil {
		return nil, err
	}
	return coordinator.appendShipContainerRebuildSteps(steps), nil
}

type startramToggleTransitionCoordinator struct {
	runtime  startramRuntime
	settings config.StartramSettings
}

func (c startramToggleTransitionCoordinator) buildWireguardSteps() ([]workflow.Step, error) {
	if c.settings.WgOn {
		return c.buildWireguardDisableSteps()
	}
	return c.buildWireguardEnableSteps()
}

func (c startramToggleTransitionCoordinator) buildWireguardDisableSteps() ([]workflow.Step, error) {
	return buildStartramToggleWireguardDisableSteps(c.runtime, c.settings)
}

func (c startramToggleTransitionCoordinator) buildWireguardEnableSteps() ([]workflow.Step, error) {
	return buildStartramToggleWireguardEnableSteps(c.runtime)
}

func (c startramToggleTransitionCoordinator) appendShipContainerRebuildSteps(steps []workflow.Step) []workflow.Step {
	return appendShipContainerRebuildSteps(steps, shipContainerRebuildRuntime{
		DeleteContainerFn: c.runtime.DeleteContainerFn,
		LoadMCFn:          c.runtime.LoadMCFn,
		LoadMinIOsFn:      c.runtime.LoadMinIOsFn,
	}, shipContainerRebuildOptions{
		piers:             c.settings.Piers,
		deletePiers:       false,
		deleteMinioClient: true,
		loadMinIOClient:   true,
		loadMinIOs:        true,
	})
}

func buildStartramToggleWireguardDisableSteps(runtime startramRuntime, settings config.StartramSettings) ([]workflow.Step, error) {
	coordinator := startramToggleRuntimeBuilder{runtime: runtime}
	steps := []workflow.Step{
		{
			Name: "update config to disable wireguard",
			Run:  coordinator.setWireguardConfig(false),
		},
		{
			Name: "stop wireguard container",
			Run:  coordinator.stopContainer("wireguard"),
		},
	}

	for _, patp := range settings.Piers {
		step := buildStartramWireguardRevertSteps(runtime, patp)
		if step != nil {
			steps = append(steps, *step)
		}
	}
	return steps, nil
}

func buildStartramWireguardRevertSteps(runtime startramRuntime, patp string) *workflow.Step {
	dockerConfig := runtime.UrbitConfFn(patp)
	if dockerConfig.Network != "wireguard" {
		return nil
	}

	patpCopy := patp
	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Type:   "urbit",
			Action: "toggle-network",
			Patp:   patpCopy,
		},
	}
	return &workflow.Step{
		Name: fmt.Sprintf("dispatch toggle-network for %s", patpCopy),
		Run:  startramWireguardRevertStep(runtime, payload),
	}
}

func startramWireguardRevertStep(runtime startramRuntime, payload structs.WsUrbitPayload) func() error {
	return func() error {
		return runtime.DispatchUrbitPayloadFn(payload)
	}
}

func buildStartramToggleWireguardEnableSteps(runtime startramRuntime) ([]workflow.Step, error) {
	coordinator := startramToggleRuntimeBuilder{runtime: runtime}
	return []workflow.Step{
		{
			Name: "update config to enable wireguard",
			Run:  coordinator.setWireguardConfig(true),
		},
		{
			Name: "start wireguard container",
			Run:  coordinator.startContainer("wireguard", "wireguard"),
		},
	}, nil
}

type startramToggleRuntimeBuilder struct {
	runtime startramRuntime
}

func (b startramToggleRuntimeBuilder) setWireguardConfig(enabled bool) func() error {
	return func() error {
		return b.runtime.UpdateConfTypedFn(config.WithWgOn(enabled))
	}
}

func (b startramToggleRuntimeBuilder) stopContainer(name string) func() error {
	return func() error {
		return b.runtime.StopContainerByNameFn(name)
	}
}

func (b startramToggleRuntimeBuilder) startContainer(name string, ctype string) func() error {
	return func() error {
		_, err := b.runtime.StartContainerFn(name, ctype)
		return err
	}
}

func HandleStartramEndpoint(endpoint string) error {
	return runStartramEndpointWithRuntime(defaultStartramRuntime(), endpoint)
}

func runStartramEndpointWithRuntime(runtime startramRuntime, endpoint string) error {
	runtime = resolveStartramRuntime(runtime)
	settings := runtime.StartramSettingsSnapshotFn()
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionEndpoint,
		startEvent:     startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionInit),
		successEvent:   startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionComplete),
		emitSuccess:    true,
		clearEvent:     startramEvent(transition.StartramTransitionEndpoint, nil),
		clearDelay:     3 * time.Second,
	},
		runStartramEndpointTransitionSteps(runtime, settings, endpoint)...)
}

func runStartramEndpointTransitionSteps(runtime startramRuntime, settings config.StartramSettings, endpoint string) []transitionStep[structs.Event] {
	coordinator := startramEndpointTransitionCoordinator{
		runtime:  runtime,
		settings: settings,
		endpoint: endpoint,
	}
	return []transitionStep[structs.Event]{
		{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionStopping),
			Run:   coordinator.stopWireguard,
		},
		{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionUnregistering),
			Run:   coordinator.unregisterAnchors,
		},
		{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionConfiguring),
			Run:   coordinator.cycleWireguardKey,
		},
		{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionFinalizing),
			Run:   coordinator.persistNewEndpoint,
		},
	}
}

type startramEndpointTransitionCoordinator struct {
	runtime  startramRuntime
	settings config.StartramSettings
	endpoint string
}

func (c startramEndpointTransitionCoordinator) stopWireguard() error {
	if !c.settings.WgOn {
		return nil
	}
	if err := c.runtime.StopContainerByNameFn("wireguard"); err != nil {
		return fmt.Errorf("stop wireguard: %w", err)
	}
	return nil
}

func (c startramEndpointTransitionCoordinator) unregisterAnchors() error {
	if !c.settings.WgRegistered {
		return nil
	}
	endpointErrs := make([]error, 0, 2*len(c.settings.Piers))
	for _, p := range c.settings.Piers {
		if err := c.runtime.SvcDeleteFn(p, "urbit"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v", p))
			endpointErrs = append(endpointErrs, fmt.Errorf("remove urbit anchor %v: %w", p, err))
		}
		if err := c.runtime.SvcDeleteFn("s3."+p, "s3"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v", p))
			endpointErrs = append(endpointErrs, fmt.Errorf("remove s3 anchor for %v: %w", p, err))
		}
	}
	if len(endpointErrs) > 0 {
		return errors.Join(endpointErrs...)
	}
	return nil
}

func (c startramEndpointTransitionCoordinator) cycleWireguardKey() error {
	return c.runtime.CycleWgKeyFn()
}

func (c startramEndpointTransitionCoordinator) persistNewEndpoint() error {
	return c.runtime.UpdateConfTypedFn(
		config.WithEndpointURL(c.endpoint),
		config.WithWgRegistered(false),
	)
}

func HandleStartramCancel(key string, reset bool) error {
	return runStartramCancelWithRuntime(defaultStartramRuntime(), key, reset)
}

func runStartramCancelWithRuntime(runtime startramRuntime, key string, reset bool) error {
	runtime = resolveStartramRuntime(runtime)
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionCancel,
		clearEvent:     startramEvent(transition.StartramTransitionCancel, nil),
		clearDelay:     3 * time.Second,
	},
		startramCancelTransitionStep{
			runtime: runtime,
			key:     key,
			reset:   reset,
		}.step(),
	)
}

type startramCancelTransitionStep struct {
	runtime startramRuntime
	key     string
	reset   bool
}

func (s startramCancelTransitionStep) step() transitionStep[structs.Event] {
	return transitionStep[structs.Event]{Run: s.run}
}

func (s startramCancelTransitionStep) run() error {
	cancelErrs := make([]error, 0, len(s.runtime.GetStartramConfigFn().Subdomains)+2)
	if s.reset {
		for _, svc := range s.runtime.GetStartramConfigFn().Subdomains {
			if err := s.runtime.SvcDeleteFn(svc.URL, svc.SvcType); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't delete service %v: %v", svc.URL, err))
				cancelErrs = append(cancelErrs, fmt.Errorf("remove service %v (%s): %w", svc.URL, svc.SvcType, err))
			}
		}
	}
	if err := startram.CancelSub(s.key); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't cancel subscription: %v", err))
		cancelErrs = append(cancelErrs, fmt.Errorf("cancel subscription: %w", err))
	}
	if err := s.runtime.CycleWgKeyFn(); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		cancelErrs = append(cancelErrs, fmt.Errorf("cycle wireguard key: %w", err))
	}
	if len(cancelErrs) > 0 {
		return errors.Join(cancelErrs...)
	}
	return nil
}

func HandleStartramReminder(remind bool) error {
	return runStartramReminderWithRuntime(defaultStartramRuntime(), remind)
}

func runStartramReminderWithRuntime(runtime startramRuntime, remind bool) error {
	runtime = resolveStartramRuntime(runtime)
	ships := runtime.ShipSettingsSnapshotFn()
	var steps []workflow.Step
	for _, patp := range ships.Piers {
		pier := patp
		steps = append(steps, workflow.Step{
			Name: fmt.Sprintf("update startram reminder for %s", pier),
			Run: func() error {
				return runtime.UpdateUrbitFeatureConfigFn(pier, func(featureConf *structs.UrbitFeatureConfig) error {
					featureConf.StartramReminder = remind
					return nil
				})
			},
		})
	}
	if joined := runOrchestrationSteps(steps, "startram reminder"); joined != nil {
		return joined
	}
	return nil
}
