package shipworkflow

import (
	"context"
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/docker/events"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/internal/seams"
	"groundseg/internal/shipstatus"
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
	EventRuntime          events.EventBroker
	TransitionEmitter     workflowTransitionFn               `runtime:"workflow" runtime_name:"transition emitter"`
	TransitionErrorPolicy transition.TransitionPublishPolicy `runtime:"workflow" runtime_name:"transition error policy"`
	Sleeper               workflowSleeperFn                  `runtime:"workflow" runtime_name:"sleeper"`
	CNAMEResolver         workflowCNAMEResolverFn            `runtime:"workflow" runtime_name:"CNAME resolver"`
	BarExit               workflowBarExitFn                  `runtime:"workflow" runtime_name:"bar exit callback"`
	dockerOrchestration.RuntimeContainerOps
	dockerOrchestration.RuntimeUrbitOps
	dockerOrchestration.RuntimeSnapshotOps
	UploadImportCoordinator workflowUploadImportCoordinatorFn `runtime:"workflow" runtime_name:"upload import coordinator"`
}

type workflowTransitionFn func(patp string, transitionType string, event string) error

type workflowSleeperFn func(time.Duration)

type workflowCNAMEResolverFn func(string) (string, error)

type workflowBarExitFn func(patp string) error

type workflowUploadImportCoordinatorFn func(context.Context, UploadImportCommand) error

var errUploadImportCoordinatorUnconfigured = fmt.Errorf("workflow runtime upload import coordinator is not configured")
var errShipStatusNotFound = shipstatus.ErrShipStatusNotFound

type workflowAliasLookupFn func(string) (string, error)

type transitionPlan[E comparable] = transitionlifecycle.LifecyclePlan[E]
type transitionStep[E comparable] = transitionlifecycle.LifecycleStep[E]

func runTransitionLifecycle[E comparable](runtime workflowRuntime, emit func(E) error, plan transitionPlan[E], steps ...transitionStep[E]) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
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
		EventRuntime:            defaultEventRuntime,
		TransitionEmitter:       defaultEmit,
		TransitionErrorPolicy:   transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		Sleeper:                 time.Sleep,
		CNAMEResolver:           net.LookupCNAME,
		RuntimeSnapshotOps:      orchestrationRuntime.RuntimeSnapshotOps,
		RuntimeContainerOps:     orchestrationRuntime.RuntimeContainerOps,
		RuntimeUrbitOps:         orchestrationRuntime.RuntimeUrbitOps,
		BarExit:                 click.BarExit,
		UploadImportCoordinator: defaultDispatch,
	}
}

func defaultWorkflowRuntime() workflowRuntime {
	return newWorkflowRuntime()
}

func resolveWorkflowRuntime(overrides ...workflowRuntime) (workflowRuntime, error) {
	runtime := defaultWorkflowRuntime()
	if len(overrides) > 0 {
		runtime = seams.Merge(runtime, overrides[0])
	}
	if err := runtime.validate(); err != nil {
		return runtime, err
	}
	return runtime, nil
}

func (runtime workflowRuntime) validate() error {
	if err := seams.NewCallbackRequirementsWithGroups("workflow").ValidateCallbacks(runtime, "workflow runtime"); err != nil {
		return seams.MissingRuntimeDependency("workflow runtime", err.Error())
	}
	return nil
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
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
	if err != nil {
		return err
	}
	runtime = resolvedRuntime
	return runtime.UploadImportCoordinator(ctx, cmd)
}

func emitWorkflowTransition(runtime workflowRuntime, patp, transitionType, event string) error {
	resolvedRuntime, _ := resolveWorkflowRuntime(runtime)
	runtime = resolvedRuntime
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
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
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
	runtime, err := resolveWorkflowRuntime(runtime)
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
	runtime, err := resolveWorkflowRuntime(defaultWorkflowRuntime())
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
	runtime, err := resolveWorkflowRuntime(defaultWorkflowRuntime())
	if err != nil {
		return err
	}
	return WaitForUrbitStop(patp, func(piers []string) (map[string]string, error) {
		return runtime.GetShipStatusFn(piers)
	}, PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return PersistUrbitConfig(patp, mutate, resolvedRuntime.UpdateUrbitFn)
}

func persistShipUrbitRuntimeConfig(patp string, mutate func(*structs.UrbitRuntimeConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitRuntimeConfigFn(patp, mutate)
}

func persistShipUrbitNetworkConfig(patp string, mutate func(*structs.UrbitNetworkConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitNetworkConfigFn(patp, mutate)
}

func persistShipUrbitScheduleConfig(patp string, mutate func(*structs.UrbitScheduleConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitScheduleConfigFn(patp, mutate)
}

func persistShipUrbitFeatureConfig(patp string, mutate func(*structs.UrbitFeatureConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitFeatureConfigFn(patp, mutate)
}

func persistShipUrbitWebConfig(patp string, mutate func(*structs.UrbitWebConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitWebConfigFn(patp, mutate)
}

func persistShipUrbitBackupConfig(patp string, mutate func(*structs.UrbitBackupConfig) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return err
	}
	return resolvedRuntime.UpdateUrbitBackupConfigFn(patp, mutate)
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate)
}

func areSubdomainsAliases(domain1, domain2 string, runtime ...workflowRuntime) (bool, error) {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return false, err
	}
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("invalid subdomain")
	}
	if startramConfig := resolvedRuntime.GetStartramConfigFn(); startramConfig.Cname != "" && domain1[firstDot+1:] == startramConfig.Cname {
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
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
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

func shipStatusNotFoundErr(patp string) error {
	return shipstatus.NotFoundErr(patp)
}

func UrbitCleanDelete(patp string) error {
	return urbitCleanDelete(patp)
}
