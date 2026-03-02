package subsystem

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"groundseg/dockerclient"
	"groundseg/internal/workflow"
	"groundseg/lifecycle"
	"groundseg/logger"
	"groundseg/orchestration"
	"groundseg/structs"
	"groundseg/transition"

	eventtypes "github.com/docker/docker/api/types/events"
)

var (
	eventBus           = make(chan structs.Event, 100)
	DisableShipRestart = false
	dockerHealthLoopMu sync.Mutex
	healthLoopRunning  bool
	healthLoopStopFn   context.CancelFunc
)

func StartDockerHealthLoops() {
	StartDockerHealthLoopsWithContext(context.Background())
}

func StartDockerHealthLoopsWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	startDockerHealthLoopWithContext(ctx)
}

func StartDockerSubsystem(ctx context.Context) {
	if err := StartDockerSubsystemWithContext(ctx); err != nil {
		return
	}
}

func StartDockerSubsystemWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	startDockerHealthLoopWithContext(ctx)
	errs := make(chan error, 2)
	go func() {
		dockerListenerWithContext(ctx)
		errs <- nil
	}()
	go func() {
		dockerSubscriptionHandler(ctx)
		errs <- nil
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return err
		}
	}
}

type dockerSubscriptionPlan interface {
	applyState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error
	afterState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error
}

type dockerStopTransitionPlan struct{}

func (dockerStopTransitionPlan) applyState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	return dockerStopTransition(rt, contName, state)
}

func (dockerStopTransitionPlan) afterState(_ dockerRoutineRuntime, _ string, _ *structs.ContainerState) error {
	return nil
}

type dockerStartTransitionPlan struct{}

func (dockerStartTransitionPlan) applyState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	return dockerStartTransition(rt, contName, state)
}

func (dockerStartTransitionPlan) afterState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	return dockerStartAfterTransition(rt, contName, state)
}

type dockerDieTransitionPlan struct{}

func (dockerDieTransitionPlan) applyState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	return dockerDieTransition(rt, contName, state)
}

func (dockerDieTransitionPlan) afterState(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	return dockerDieAfterTransition(rt, contName, state)
}

func dockerSubscriptionPlanForAction(action transition.DockerAction) (dockerSubscriptionPlan, bool) {
	switch action {
	case transition.DockerActionStop:
		return dockerStopTransitionPlan{}, true
	case transition.DockerActionStart:
		return dockerStartTransitionPlan{}, true
	case transition.DockerActionDie:
		return dockerDieTransitionPlan{}, true
	default:
		return nil, false
	}
}

// dockerListenerWithContext runs until ctx is canceled.
func dockerListenerWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	cli, err := dockerclient.New()
	if err != nil {
		logger.Error(fmt.Sprintf("Error initializing Docker client: %v", err))
		return
	}
	messages, errs := cli.Events(ctx, eventtypes.ListOptions{})
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker event listener")
			return
		case event := <-messages:
			if ctx.Err() != nil {
				return
			}
			// Convert the Docker event to our custom event and send it to the EventBus.
			select {
			case eventBus <- structs.Event{Type: string(event.Action), Data: event}:
			case <-ctx.Done():
				return
			}
		case err := <-errs:
			if err == nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			logger.Error(fmt.Sprintf("Docker event error: %v", err))
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func dockerSubscriptionHandler(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	rt := newDockerRoutineRuntime()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker subscription handler")
			return
		case event := <-eventBus:
			dockerEvent, ok := event.Data.(eventtypes.Message)
			if !ok {
				logger.Error("Failed to assert Docker event data type")
				continue
			}
			contName := dockerEvent.Actor.Attributes["name"]
			action := transition.DockerAction(dockerEvent.Action)
			plan, ok := dockerSubscriptionPlanForAction(action)
			if !ok {
				logger.Debug(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
				continue
			}
			if err := runDockerTransitionWorkflow(rt, contName, plan); err != nil {
				logger.Warn(fmt.Sprintf("Docker transition failed for %s: %v", contName, err))
				continue
			}
		}
	}
}

func runDockerTransitionWorkflow(rt dockerRoutineRuntime, contName string, plan dockerSubscriptionPlan) error {
	var containerState *structs.ContainerState
	return orchestration.RunStructuredWorkflow(
		orchestration.WorkflowPhases{
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("apply-state"),
					Run: func() error {
						state, err := updateContainerTransition(rt, contName, func(state *structs.ContainerState) error {
							return plan.applyState(rt, contName, state)
						})
						if err != nil {
							return err
						}
						if state == nil {
							return nil
						}
						containerState = state
						return nil
					},
				},
				{
					Phase: lifecycle.Phase("post-state"),
					Run: func() error {
						if containerState == nil {
							return nil
						}
						return plan.afterState(rt, contName, containerState)
					},
				},
			},
		},
		orchestration.WorkflowCallbacks{
			OnSuccess: func() {
				if containerState == nil {
					return
				}
				if err := publishDockerTransition(rt); err != nil {
					logger.Warn(fmt.Sprintf("Docker broadcast failed for %s: %v", contName, err))
				}
			},
		},
	)
}

func publishDockerTransition(rt dockerRoutineRuntime) error {
	return rt.broadcastOps.broadcastClientsFn()
}

func dockerStopTransition(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	logger.Infof("Docker: %s stopped", contName)
	state.ActualStatus = string(transition.ContainerStatusStopped)
	if rt.recovery.stopTransitionRestartFn == nil {
		return nil
	}
	if err := rt.recovery.stopTransitionRestartFn(rt, contName, state); err != nil {
		return err
	}
	return nil
}

func dockerStartTransition(_ dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	logger.Infof("Docker: %s started", contName)
	state.ActualStatus = string(transition.ContainerStatusRunning)
	return nil
}

func dockerStartAfterTransition(rt dockerRoutineRuntime, contName string, _ *structs.ContainerState) error {
	if contName != string(transition.ContainerTypeWireguard) {
		return nil
	}
	if err := rt.broadcastOps.updateWgOnFn(true); err != nil {
		return err
	}
	return rt.broadcastOps.setStartramRunningFn(true)
}

func dockerDieTransition(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	logger.Warnf("Docker: %s died!", contName)
	state.ActualStatus = string(transition.ContainerStatusDied)
	if state.Type != string(transition.ContainerTypeVere) {
		return nil
	}
	if err := rt.runtimeOps.LoadUrbitConfigFn(contName); err != nil {
		return fmt.Errorf("failed to load config for %s: %w", contName, err)
	}
	conf := rt.runtimeOps.UrbitConfFn(contName)
	if conf.DisableShipRestarts {
		logger.Infof("Leaving %s container alone after death due to DisableShipRestarts=true", contName)
		state.DesiredStatus = string(transition.ContainerStatusStopped)
	}
	return nil
}

func dockerDieAfterTransition(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	if state.Type != string(transition.ContainerTypeVere) {
		return nil
	}
	rt.runtimeOps.ClearLusCodeFn(contName)
	if state.DesiredStatus == string(transition.ContainerStatusDied) || state.DesiredStatus == string(transition.ContainerStatusStopped) {
		logger.Infof("Ship desired status: %s", state.DesiredStatus)
		return nil
	}
	return scheduleShipRestart(rt, contName, state.Type)
}

func scheduleShipRestart(rt dockerRoutineRuntime, containerName, containerType string) error {
	logger.Infof("Attempting to restart ship %s after death", containerName)
	if rt.recovery.restartAfterDeathFn == nil {
		return nil
	}
	return rt.recovery.restartAfterDeathFn(rt, containerName, containerType)
}

func defaultStopTransitionRestart(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
	if state.DesiredStatus == string(transition.ContainerStatusStopped) {
		return nil
	}
	_, err := rt.runtimeOps.StartContainerFn(contName, state.Type)
	return err
}

func defaultRestartAfterDeath(rt dockerRoutineRuntime, containerName, containerType string) error {
	go func(name, ctype string) {
		rt.timer.sleepFn(rt.recovery.restartDelay)
		_, err := rt.runtimeOps.StartContainerFn(name, ctype)
		if err != nil {
			logger.Errorf("Failed to restart %s after death: %v", name, err)
			return
		}
		logger.Infof("Successfully restarted %s after death", name)
	}(containerName, containerType)
	return nil
}

func defaultRecoverWireguardAfter502(rt dockerRoutineRuntime, settings dockerCheck502Settings) error {
	if settings.Disable502 {
		return nil
	}
	return rt.wireguardOps.recoverWireguardFn(settings.Piers, false)
}

func updateContainerTransition(rt dockerRoutineRuntime, contName string, mutate func(*structs.ContainerState) error) (*structs.ContainerState, error) {
	containerState, exists := rt.runtimeOps.GetContainerStateFn()[contName]
	if !exists {
		return nil, nil
	}

	if err := mutate(&containerState); err != nil {
		return nil, err
	}
	rt.runtimeOps.UpdateContainerStateFn(contName, containerState)
	return &containerState, nil
}

// loop to make sure ships are reachable
// if 502 2x in 2 min, restart wg container
func Check502Loop() {
	Check502LoopWithContext(context.Background())
}

func Check502LoopWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	check502Loop(ctx, newDockerRoutineRuntime())
}

func check502Loop(ctx context.Context, rt dockerRoutineRuntime) {
	if ctx == nil {
		ctx = context.Background()
	}
	status := make(map[string]int)
	rt.timer.sleepFn(rt.recovery.check502InitialDelay)
	for {
		if ctx.Err() != nil {
			return
		}
		rt.timer.sleepFn(rt.recovery.check502PollDelay)
		if ctx.Err() != nil {
			return
		}
		threshold := rt.recovery.check502ConsecutiveFailures
		if threshold < 1 {
			threshold = 1
		}
		settings := rt.runtimeOps.Check502SettingsSnapshotFn()
		pierStatus, err := rt.runtimeOps.GetShipStatusFn(settings.Piers)
		if err != nil {
			logger.Errorf("Couldn't get pier status: %v", err)
			continue
		}
		for _, pier := range settings.Piers {
			if ctx.Err() != nil {
				return
			}
			err := rt.runtimeOps.LoadUrbitConfigFn(pier)
			if err != nil {
				logger.Errorf("Error loading %s config: %v", pier, err)
				continue
			}
			shipConf := rt.runtimeOps.UrbitConfFn(pier)
			pierNetwork, err := rt.runtimeOps.GetContainerNetworkFn(pier)
			if err != nil {
				logger.Warnf("Couldn't get network for %v: %v", pier, err)
				continue
			}
			turnedOn := false
			if transition.IsContainerUpStatus(pierStatus[pier]) {
				turnedOn = true
			}
			if turnedOn && pierNetwork != "default" && settings.WgOn {
				if _, err := rt.runtimeOps.GetLusCodeFn(pier); err != nil {
					logger.Warnf("%v is not booted yet, skipping", pier)
					continue
				}
				resp, err := rt.httpOps.getFn("https://" + shipConf.WgURL)
				if err != nil {
					logger.Errorf("Error remote polling %v: %v", pier, err)
					continue
				}
				resp.Body.Close()
				logger.Debugf("%v 502 check: %v", pier, resp.StatusCode)
				if resp.StatusCode == http.StatusBadGateway {
					logger.Warnf("Got 502 response for %v", pier)
					status[pier]++
					if status[pier] < threshold {
						continue
					}
					if settings.Disable502 {
						delete(status, pier)
						continue
					}
					logger.Warnf("502 strike %d/%d for %v", status[pier], threshold, pier)
					if err := rt.recovery.recoverWireguardAfter502Fn(rt, settings); err != nil {
						logger.Errorf("Wireguard fleet recovery failed: %v", err)
					}
					// remove from map after recovery attempt
					delete(status, pier)
				} else if _, found := status[pier]; found {
					// if not 502 and pier is in status map, remove it
					delete(status, pier)
				}
			}
		}
	}
}

func GracefulShipExit() error {
	return gracefulShipExit(newDockerRoutineRuntime())
}

func gracefulShipExit(rt dockerRoutineRuntime) error {
	DisableShipRestart = true
	defer func() {
		DisableShipRestart = false
	}()
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := rt.runtimeOps.GetShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	piers := rt.runtimeOps.ShipSettingsSnapshotFn().Piers
	pierStatus, err := rt.runtimeOps.GetShipStatusFn(piers)
	if err != nil {
		return fmt.Errorf("Failed to retrieve ship information: %w", err)
	}
	steps := []workflow.Step{}
	for patp, status := range pierStatus {
		if transition.IsContainerUpStatus(status) {
			pirate := patp
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("stop %s with |exit for daemon restart", pirate),
				Run:  func() error { return rt.systemOps.barExitFn(pirate) },
			})
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("wait for %s status during graceful exit", pirate),
				Run: func() error {
					for {
						status, err := getShipRunningStatus(pirate)
						if err != nil {
							return fmt.Errorf("failed to poll %s status during graceful exit: %w", pirate, err)
						}
						logger.Debugf("%s", status)
						if !transition.IsContainerUpStatus(status) {
							return nil
						}
						rt.timer.sleepFn(1 * time.Second)
					}
				},
			})
		}
	}
	if joined := workflow.Join(steps, func(err error) {
		logger.Errorf("one or more ships failed graceful shutdown: %v", err)
	}); joined != nil {
		return fmt.Errorf("one or more ships failed graceful shutdown: %w", joined)
	}
	return nil
}

func startDockerHealthLoopWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	dockerHealthLoopMu.Lock()
	defer dockerHealthLoopMu.Unlock()
	if healthLoopRunning {
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	healthLoopRunning = true
	healthLoopStopFn = cancel

	go func() {
		Check502LoopWithContext(loopCtx)
		dockerHealthLoopMu.Lock()
		healthLoopRunning = false
		dockerHealthLoopMu.Unlock()
	}()
}
