package shipworkflow

import (
	"context"
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/events"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/internal/seams"
	"groundseg/internal/transitionlifecycle"
	"groundseg/lifecycle"
	"groundseg/structs"
	"groundseg/transition"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

type workflowRuntime struct {
	EventRuntime              events.EventBroker
	TransitionEmitter         workflowTransitionFn
	TransitionErrorPolicy     transition.TransitionPublishPolicy
	Sleeper                  workflowSleeperFn
	CNAMEResolver            workflowCNAMEResolverFn
	BarExit                  workflowBarExitFn
	dockerOrchestration.RuntimeTransitionOps
	UploadImportCoordinator   workflowUploadImportCoordinatorFn
}

type workflowTransitionFn func(patp string, transitionType string, event string) error

type workflowSleeperFn func(time.Duration)

type workflowCNAMEResolverFn func(string) (string, error)

type workflowBarExitFn func(patp string) error

type workflowUploadImportCoordinatorFn func(context.Context, UploadImportCommand) error

var errUploadImportCoordinatorUnconfigured = fmt.Errorf("workflow runtime upload import coordinator is not configured")
var errShipStatusNotFound = errors.New("ship status doesn't exist")

type workflowAliasLookupFn func(string) (string, error)

type transitionPlan[E comparable] = transitionlifecycle.LifecyclePlan[E]
type transitionStep[E comparable] = transitionlifecycle.LifecycleStep[E]

func runTransitionLifecycle[E comparable](runtime workflowRuntime, emit func(E) error, plan transitionPlan[E], steps ...transitionStep[E]) error {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime)
	if err != nil {
		return err
	}
	runtime = resolvedRuntime
	return transitionlifecycle.RunLifecycle(
		transitionlifecycle.Runtime[E]{
			Emit:  emit,
			Sleep: runtime.Sleeper,
		},
		plan,
		steps...,
	)
}

func runUrbitTransition(patp string, transitionType string, plan transitionPlan[string], steps ...transitionStep[string]) error {
	runtime := defaultWorkflowRuntime()
	return runTransitionLifecycle[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransition(runtime, patp, transitionType, event)
		},
		plan,
		steps...,
	)
}

func newWorkflowRuntime() workflowRuntime {
	defaultEventRuntime := events.DefaultEventRuntime()
	defaultEmit := func(patp, transitionType, event string) error {
		return defaultEventRuntime.PublishUrbitTransition(context.Background(), structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
	}
	defaultDispatch := func(ctx context.Context, cmd UploadImportCommand) error {
		return unconfiguredUploadImportCoordinator(ctx, cmd)
	}
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return workflowRuntime{
		EventRuntime:              defaultEventRuntime,
		TransitionEmitter:         defaultEmit,
		TransitionErrorPolicy:     transition.TransitionPublishStrict,
		Sleeper:                  time.Sleep,
		CNAMEResolver:            net.LookupCNAME,
		BarExit:                  click.BarExit,
		RuntimeTransitionOps:      orchestrationRuntime.RuntimeTransitionOps,
		UploadImportCoordinator:   defaultDispatch,
	}
}

func defaultWorkflowRuntime() workflowRuntime {
	return newWorkflowRuntime()
}

func resolveWorkflowRuntime(overrides ...workflowRuntime) workflowRuntime {
	if len(overrides) == 0 {
		return defaultWorkflowRuntime()
	}
	return withDefaultsWorkflowRuntime(overrides[0])
}

func resolveWorkflowRuntimeWithDefaults(overrides ...workflowRuntime) (workflowRuntime, error) {
	runtime := resolveWorkflowRuntime(overrides...)
	if err := runtime.validate(); err != nil {
		return runtime, err
	}
	return runtime, nil
}

func (runtime workflowRuntime) validate() error {
	missing := make([]string, 0, 8)
	if runtime.TransitionEmitter == nil {
		missing = append(missing, "transition emitter")
	}
	if runtime.TransitionErrorPolicy == "" {
		missing = append(missing, "transition error policy")
	}
	if runtime.Sleeper == nil {
		missing = append(missing, "sleeper")
	}
	if runtime.CNAMEResolver == nil {
		missing = append(missing, "CNAME resolver")
	}
	if runtime.BarExit == nil {
		missing = append(missing, "bar exit callback")
	}
	if runtime.UploadImportCoordinator == nil {
		missing = append(missing, "upload import coordinator")
	}
	if runtime.UpdateUrbitFn == nil {
		missing = append(missing, "update urbit callback")
	}
	if runtime.UpdateUrbitSectionFn == nil {
		missing = append(missing, "update urbit section callback")
	}
	if runtime.GetShipStatusFn == nil {
		missing = append(missing, "ship status callback")
	}
	if runtime.DeleteContainerFn == nil {
		missing = append(missing, "delete container callback")
	}
	if runtime.StopContainerByNameFn == nil {
		missing = append(missing, "stop container callback")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("workflow runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

type UploadImportCommand struct {
	ArchivePath string
	Filename    string
	Patp        string
	Remote      bool
	Fix         bool
	CustomDrive string
}

type UploadImportCoordinator func(context.Context, UploadImportCommand) error

func unconfiguredUploadImportCoordinator(context.Context, UploadImportCommand) error {
	return errUploadImportCoordinatorUnconfigured
}

func DispatchUploadImport(ctx context.Context, cmd UploadImportCommand) error {
	return DispatchUploadImportWithCoordinator(nil, ctx, cmd)
}

func DispatchUploadImportWithCoordinator(coordinator UploadImportCoordinator, ctx context.Context, cmd UploadImportCommand) error {
	runtime := defaultWorkflowRuntime()
	if coordinator != nil {
		runtime.UploadImportCoordinator = workflowUploadImportCoordinatorFn(coordinator)
	}
	return dispatchUploadImport(runtime, ctx, cmd)
}

func dispatchUploadImport(runtime workflowRuntime, ctx context.Context, cmd UploadImportCommand) error {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime)
	if err != nil {
		return err
	}
	runtime = resolvedRuntime
	if runtime.UploadImportCoordinator == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	return runtime.UploadImportCoordinator(ctx, cmd)
}

func emitWorkflowTransition(runtime workflowRuntime, patp, transitionType, event string) error {
	if runtime.TransitionEmitter == nil {
		return fmt.Errorf("workflow runtime transition emitter is not configured")
	}
	if err := runtime.TransitionEmitter(patp, transitionType, event); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("publish urbit transition for %s", patp),
			err,
			runtime.TransitionErrorPolicy,
		)
	}
	return nil
}

func PublishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) {
	publishTransition(defaultWorkflowRuntime(), publish, event, clear, clearDelay)
}

func publishTransition[T any](runtime workflowRuntime, publish func(T), event T, clear T, clearDelay time.Duration) {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime)
	if err != nil {
		resolvedRuntime = defaultWorkflowRuntime()
	}
	runtime = resolvedRuntime
	publish(event)
	if clearDelay > 0 {
		runtime.Sleeper(clearDelay)
	}
	publish(clear)
}

func RunTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	runtime := defaultWorkflowRuntime()
	return runTransitionLifecycle[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransition(runtime, patp, transitionType, event)
		},
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   startEvent,
			SuccessEvent: successEvent,
			EmitSuccess:  successEvent != "",
			ErrorEvent:   func(_ error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   clearDelay,
		},
		transitionStep[string]{
			Run: operation,
		},
	)
}

func runTransitionedOperationWithRuntime(
	runtime workflowRuntime,
	patp,
	transitionType,
	startEvent,
	successEvent string,
	clearDelay time.Duration,
	operation func() error,
) error {
	runtime, err := resolveWorkflowRuntimeWithDefaults(runtime)
	if err != nil {
		return err
	}
	return runTransitionLifecycle[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransition(runtime, patp, transitionType, event)
		},
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   startEvent,
			SuccessEvent: successEvent,
			EmitSuccess:  successEvent != "",
			ErrorEvent:   func(_ error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   clearDelay,
		},
		transitionStep[string]{
			Run: operation,
		},
	)
}

func waitForDeskState(patp, desk, expectedState string, shouldMatch bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return PollWithTimeout(ctx, 5*time.Second, func() (bool, error) {
		status, err := click.GetDesk(patp, desk, true)
		if err != nil {
			return false, fmt.Errorf("get %s desk status for %s: %w", desk, patp, err)
		}
		if shouldMatch {
			return status == expectedState, nil
		}
		return status != expectedState, nil
	})
}

func runDeskTransition(patp, transitionType string, operation func() error) error {
	return RunTransitionedOperation(patp, transitionType, "loading", "success", 3*time.Second, operation)
}

func runPhaseWorkflow(
	patp string,
	transitionType string,
	successEvent string,
	clearDelay time.Duration,
	steps ...lifecycle.Step,
) error {
	runtime, err := resolveWorkflowRuntimeWithDefaults(defaultWorkflowRuntime())
	if err != nil {
		return err
	}
	phaseSteps := make([]transitionStep[string], 0, len(steps))
	for _, step := range steps {
		phaseSteps = append(phaseSteps, transitionStep[string]{
			Event: string(step.Phase),
			Run:   step.Run,
		})
	}

	return runTransitionLifecycle[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransition(runtime, patp, transitionType, event)
		},
		transitionPlan[string]{
			SuccessEvent: successEvent,
			EmitSuccess:  successEvent != "",
			ErrorEvent:   func(_ error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   clearDelay,
		},
		phaseSteps...,
	)
}

func WaitComplete(patp string) error {
	runtime, err := resolveWorkflowRuntimeWithDefaults(defaultWorkflowRuntime())
	if err != nil {
		return err
	}
	return WaitForUrbitStop(patp, func(piers []string) (map[string]string, error) {
		return runtime.GetShipStatusFn(piers)
	}, PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime...)
	if err != nil {
		return err
	}
	return PersistUrbitConfig(patp, mutate, resolvedRuntime.UpdateUrbitFn)
}

func persistShipUrbitConfig[T any](patp string, section dockerOrchestration.UrbitConfigSection, mutate func(*T) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitSectionFn(patp, section, any(mutate))
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate)
}

func areSubdomainsAliases(domain1, domain2 string, runtime ...workflowRuntime) (bool, error) {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime...)
	if err != nil {
		return false, err
	}
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("invalid subdomain")
	}
	if config.GetStartramConfig().Cname != "" && domain1[firstDot+1:] == config.GetStartramConfig().Cname {
		return true, nil
	}
		cname1, err := resolvedRuntime.CNAMEResolver(domain1)
		if err != nil {
			return false, fmt.Errorf("lookup CNAME for %s: %w", domain1, err)
		}
		cname2, err := resolvedRuntime.CNAMEResolver(domain2)
		if err != nil {
			return false, fmt.Errorf("lookup CNAME for %s: %w", domain2, err)
		}
	return cname1 == cname2, nil
}

func AreSubdomainsAliases(domain1, domain2 string) (bool, error) {
	return areSubdomainsAliases(domain1, domain2)
}

func AreSubdomainsAliasesWithLookup(
	lookupAlias func(string) (string, error),
	domain1 string,
	domain2 string,
) (bool, error) {
	runtime := workflowRuntime{
		CNAMEResolver: workflowCNAMEResolverFn(lookupAlias),
	}
	return areSubdomainsAliases(domain1, domain2, runtime)
}

func urbitCleanDelete(patp string, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntimeWithDefaults(runtime...)
	if err != nil {
		return err
	}
	stopErrs := make([]error, 0, 2)
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := resolvedRuntime.GetShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("failed to get ship status for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", shipStatusNotFoundErr(patp)
		}
		return status, nil
	}
	status, err := getShipRunningStatus(patp)
		if err == nil {
			if strings.Contains(status, "Up") {
				if err := resolvedRuntime.BarExit(patp); err != nil {
					stopErrs = append(stopErrs, fmt.Errorf("failed to stop %s with |exit: %w", patp, err))
					if err = resolvedRuntime.StopContainerByNameFn(patp); err != nil {
						stopErrs = append(stopErrs, fmt.Errorf("failed to stop container %s by name: %w", patp, err))
				}
			}
		}
		for {
			status, err := getShipRunningStatus(patp)
			if err != nil {
				break
			}
			zap.L().Debug(fmt.Sprintf("%s", status))
				if !strings.Contains(status, "Up") {
					break
				}
				resolvedRuntime.Sleeper(1 * time.Second)
			}
		}
	deleteErr := resolvedRuntime.DeleteContainerFn(patp)
	if len(stopErrs) == 0 {
		return deleteErr
	}
	stopErr := errors.Join(stopErrs...)
	if deleteErr == nil {
		return stopErr
	}
	return errors.Join(stopErr, fmt.Errorf("failed to delete ship container for %s: %w", patp, deleteErr))
}

func withDefaultsWorkflowRuntime(runtime workflowRuntime) workflowRuntime {
	base := defaultWorkflowRuntime()
	resolvedRuntime := seams.Merge(base, runtime)
	if resolvedRuntime.UpdateUrbitFn == nil {
		resolvedRuntime.UpdateUrbitFn = config.UpdateUrbit
	}
	if resolvedRuntime.UpdateUrbitSectionFn == nil {
		resolvedRuntime.UpdateUrbitSectionFn = config.UpdateUrbitSectionConfig
	}
	if resolvedRuntime.TransitionEmitter == nil {
		eventRuntime := resolvedRuntime.EventRuntime
		resolvedRuntime.TransitionEmitter = workflowTransitionFn(func(patp, transitionType, event string) error {
			return eventRuntime.PublishUrbitTransition(context.Background(), structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
		})
	}
	if resolvedRuntime.TransitionErrorPolicy == "" {
		resolvedRuntime.TransitionErrorPolicy = transition.TransitionPublishStrict
	}
	if resolvedRuntime.Sleeper == nil {
		resolvedRuntime.Sleeper = time.Sleep
	}
	if resolvedRuntime.CNAMEResolver == nil {
		resolvedRuntime.CNAMEResolver = net.LookupCNAME
	}
	if resolvedRuntime.BarExit == nil {
		resolvedRuntime.BarExit = click.BarExit
	}
	if resolvedRuntime.UploadImportCoordinator == nil {
		resolvedRuntime.UploadImportCoordinator = unconfiguredUploadImportCoordinator
	}
	return resolvedRuntime
}

func shipStatusNotFoundErr(patp string) error {
	return fmt.Errorf("%w: %s", errShipStatusNotFound, patp)
}

func UrbitCleanDelete(patp string) error {
	return urbitCleanDelete(patp)
}
