package subsystem

import (
	"fmt"

	"groundseg/lifecycle"
	"groundseg/logger"
	"groundseg/orchestration"
	"groundseg/structs"
	"groundseg/transition"
)

type dockerSubscriptionPlan struct {
	applyState func(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error
	afterState func(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error
}

var dockerSubscriptionPlans = map[transition.DockerAction]dockerSubscriptionPlan{
	transition.DockerActionStop: {
		applyState: func(rt dockerRoutineRuntime, contName string, state *structs.ContainerState) error {
			return dockerStopTransition(rt, contName, state)
		},
	},
	transition.DockerActionStart: {
		applyState: dockerStartTransition,
		afterState: dockerStartAfterTransition,
	},
	transition.DockerActionDie: {
		applyState: dockerDieTransition,
		afterState: dockerDieAfterTransition,
	},
}

func dockerSubscriptionPlanForAction(action transition.DockerAction) (dockerSubscriptionPlan, bool) {
	plan, ok := dockerSubscriptionPlans[action]
	return plan, ok
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
				{
					Phase: lifecycle.Phase("broadcast"),
					Run: func() error {
						if containerState == nil {
							return nil
						}
						return publishDockerTransition(rt)
					},
				},
			},
		},
		orchestration.WorkflowCallbacks{},
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
	if err := rt.transitionOps.LoadUrbitConfigFn(contName); err != nil {
		return fmt.Errorf("failed to load config for %s: %w", contName, err)
	}
	conf := rt.transitionOps.UrbitConfFn(contName)
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
	rt.transitionOps.ClearLusCodeFn(contName)
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

func updateContainerTransition(rt dockerRoutineRuntime, contName string, mutate func(*structs.ContainerState) error) (*structs.ContainerState, error) {
	containerState, exists := rt.transitionOps.GetContainerStateFn()[contName]
	if !exists {
		return nil, nil
	}

	if err := mutate(&containerState); err != nil {
		return nil, err
	}
	rt.transitionOps.UpdateContainerStateFn(contName, containerState)
	return &containerState, nil
}
