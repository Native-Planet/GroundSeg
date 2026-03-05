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
	"groundseg/internal/shipstatus"
	"groundseg/internal/transitionlifecycle"
	"groundseg/lifecycle"
	"groundseg/shipworkflow/adapters/lifecyclebridge"
	"groundseg/shipworkflow/lifecyclewait"
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
	workflowShipOps
	workflowConfigOps
	UploadImportCoordinator workflowUploadImportCoordinatorFn `runtime:"workflow" runtime_name:"upload import coordinator"`
}

type workflowShipOps struct {
	GetShipStatusFn       func([]string) (map[string]string, error) `runtime:"workflow" runtime_name:"get ship status callback"`
	StopContainerByNameFn func(name string) error                   `runtime:"workflow" runtime_name:"stop container callback"`
	DeleteContainerFn     func(name string) error                   `runtime:"workflow" runtime_name:"delete container callback"`
}

type workflowConfigOps struct {
	UpdateUrbitFn        func(string, func(*structs.UrbitDocker) error) error                                              `runtime:"workflow" runtime_name:"update urbit callback"`
	UpdateUrbitSectionFn func(patp string, section dockerOrchestration.UrbitConfigSection, mutateFn func(any) error) error `runtime:"workflow" runtime_name:"update urbit section callback"`
	GetStartramConfigFn  func() structs.StartramRetrieve                                                                   `runtime:"workflow" runtime_name:"startram config callback"`
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
		return fmt.Errorf("prepare workflow runtime for transition lifecycle: %w", err)
	}
	return runTransitionLifecycleResolved(resolvedRuntime, emit, plan, steps...)
}

func runTransitionLifecycleResolved[E comparable](runtime workflowRuntime, emit func(E) error, plan transitionPlan[E], steps ...transitionStep[E]) error {
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
	runtime, err := resolveWorkflowRuntime(newWorkflowRuntime())
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for urbit transition %s on %s: %w", transitionType, patp, err)
	}
	return runTransitionLifecycleResolved[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransitionResolved(runtime, patp, transitionType, event)
		},
		plan,
		steps...,
	)
}

func newWorkflowRuntime() workflowRuntime {
	defaultEventBroker := events.NewRuntimeBoundBroker(events.DefaultEventRuntime())
	defaultEmit := func(patp, transitionType, event string) error {
		return defaultEventBroker.PublishUrbitTransition(context.Background(), structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
	}
	defaultDispatch := func(ctx context.Context, cmd UploadImportCommand) error {
		return unconfiguredUploadImportCoordinator(ctx, cmd)
	}
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return workflowRuntime{
		EventRuntime:          defaultEventBroker,
		TransitionEmitter:     defaultEmit,
		TransitionErrorPolicy: transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		Sleeper:               time.Sleep,
		CNAMEResolver:         net.LookupCNAME,
		workflowShipOps: workflowShipOps{
			GetShipStatusFn:       orchestrationRuntime.GetShipStatusFn,
			StopContainerByNameFn: orchestrationRuntime.StopContainerByNameFn,
			DeleteContainerFn:     orchestrationRuntime.DeleteContainerFn,
		},
		workflowConfigOps: workflowConfigOps{
			UpdateUrbitFn:        orchestrationRuntime.UpdateUrbitFn,
			UpdateUrbitSectionFn: orchestrationRuntime.UpdateUrbitSectionFn,
			GetStartramConfigFn:  orchestrationRuntime.GetStartramConfigFn,
		},
		BarExit:                 click.BarExit,
		UploadImportCoordinator: defaultDispatch,
	}
}

func resolveWorkflowRuntime(overrides ...workflowRuntime) (workflowRuntime, error) {
	return seams.ResolveRuntime(newWorkflowRuntime(), workflowRuntime.validate, overrides...)
}

func (runtime workflowRuntime) validate() error {
	missing := make([]string, 0, 8)
	if runtime.TransitionEmitter == nil {
		missing = append(missing, "transition emitter")
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
	if runtime.GetShipStatusFn == nil {
		missing = append(missing, "ship status callback")
	}
	if runtime.StopContainerByNameFn == nil {
		missing = append(missing, "stop container callback")
	}
	if runtime.DeleteContainerFn == nil {
		missing = append(missing, "delete container callback")
	}
	if runtime.UpdateUrbitFn == nil {
		missing = append(missing, "update urbit callback")
	}
	if runtime.UpdateUrbitSectionFn == nil {
		missing = append(missing, "update urbit section callback")
	}
	if runtime.GetStartramConfigFn == nil {
		missing = append(missing, "startram config callback")
	}
	if runtime.UploadImportCoordinator == nil {
		missing = append(missing, "upload import coordinator")
	}
	if len(missing) > 0 {
		return seams.MissingRuntimeDependency("workflow runtime", strings.Join(missing, ", "))
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
	runtime := newWorkflowRuntime()
	if coordinator != nil {
		runtime.UploadImportCoordinator = workflowUploadImportCoordinatorFn(coordinator)
	}
	return dispatchUploadImport(runtime, ctx, cmd)
}

func dispatchUploadImport(runtime workflowRuntime, ctx context.Context, cmd UploadImportCommand) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for upload import dispatch: %w", err)
	}
	runtime = resolvedRuntime
	if err := runtime.UploadImportCoordinator(ctx, cmd); err != nil {
		return fmt.Errorf("dispatch upload import for %s: %w", cmd.Patp, err)
	}
	return nil
}

func emitWorkflowTransition(runtime workflowRuntime, patp, transitionType, event string) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime: %w", err)
	}
	return emitWorkflowTransitionResolved(resolvedRuntime, patp, transitionType, event)
}

func emitWorkflowTransitionResolved(runtime workflowRuntime, patp, transitionType, event string) error {
	if err := runtime.TransitionEmitter(patp, transitionType, event); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("publish urbit transition for %s", patp),
			err,
			runtime.TransitionErrorPolicy,
		)
	}
	return nil
}

func PublishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) error {
	return publishTransition(newWorkflowRuntime(), publish, event, clear, clearDelay)
}

func publishTransition[T any](runtime workflowRuntime, publish func(T), event T, clear T, clearDelay time.Duration) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for transition publish: %w", err)
	}
	runtime = resolvedRuntime
	publish(event)
	if clearDelay > 0 {
		runtime.Sleeper(clearDelay)
	}
	publish(clear)
	return nil
}

func RunTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	runtime, err := resolveWorkflowRuntime(newWorkflowRuntime())
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for transitioned operation %s on %s: %w", transitionType, patp, err)
	}
	return runTransitionLifecycleResolved[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransitionResolved(runtime, patp, transitionType, event)
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

func RunTransitionedOperationWithRuntime(
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
		return fmt.Errorf("prepare workflow runtime for transitioned operation %s on %s: %w", transitionType, patp, err)
	}
	return runTransitionLifecycleResolved[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransitionResolved(runtime, patp, transitionType, event)
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
	return lifecyclebridge.PollWithTimeout(ctx, 5*time.Second, func() (bool, error) {
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
	runtime, err := resolveWorkflowRuntime(newWorkflowRuntime())
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for phase workflow %s on %s: %w", transitionType, patp, err)
	}
	phaseSteps := make([]transitionStep[string], 0, len(steps))
	for _, step := range steps {
		phaseSteps = append(phaseSteps, transitionStep[string]{
			Event: string(step.Phase),
			Run:   step.Run,
		})
	}

	return runTransitionLifecycleResolved[string](
		runtime,
		func(event string) error {
			return emitWorkflowTransitionResolved(runtime, patp, transitionType, event)
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
	return WaitCompleteWithRuntime(workflowRuntime{}, patp)
}

func WaitCompleteWithRuntime(runtime workflowRuntime, patp string) error {
	runtime, err := resolveWorkflowRuntime(runtime)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for wait complete on %s: %w", patp, err)
	}
	return lifecyclewait.WaitForUrbitStop(patp, func(piers []string) (map[string]string, error) {
		return runtime.GetShipStatusFn(piers)
	}, lifecyclebridge.PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for persist ship config %s: %w", patp, err)
	}
	if err := lifecyclebridge.PersistUrbitConfig(patp, mutate, resolvedRuntime.UpdateUrbitFn); err != nil {
		return fmt.Errorf("persist ship config %s: %w", patp, err)
	}
	return nil
}

func persistShipUrbitSection(patp string, section dockerOrchestration.UrbitConfigSection, mutateFn func(any) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for persist ship section %s (%s): %w", patp, section, err)
	}
	if err := resolvedRuntime.UpdateUrbitSectionFn(patp, section, mutateFn); err != nil {
		return fmt.Errorf("persist ship section %s (%s): %w", patp, section, err)
	}
	return nil
}

func persistShipUrbitSectionConfig[T any](patp string, section dockerOrchestration.UrbitConfigSection, mutate func(*T) error, runtime ...workflowRuntime) error {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return fmt.Errorf("prepare workflow runtime for persist ship section config %s (%s): %w", patp, section, err)
	}
	if err := persistShipUrbitSection(patp, section, config.AdaptUrbitSectionMutation(mutate), resolvedRuntime); err != nil {
		return fmt.Errorf("persist ship section config %s (%s): %w", patp, section, err)
	}
	return nil
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return PersistUrbitConfigValueWithRuntime(workflowRuntime{}, patp, mutate)
}

func PersistUrbitConfigValueWithRuntime(runtime workflowRuntime, patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate, runtime)
}

func areSubdomainsAliases(domain1, domain2 string, runtime ...workflowRuntime) (bool, error) {
	resolvedRuntime, err := resolveWorkflowRuntime(runtime...)
	if err != nil {
		return false, fmt.Errorf("prepare workflow runtime for alias comparison %s/%s: %w", domain1, domain2, err)
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
	return AreSubdomainsAliasesWithRuntime(workflowRuntime{}, domain1, domain2)
}

func AreSubdomainsAliasesWithRuntime(runtime workflowRuntime, domain1, domain2 string) (bool, error) {
	return areSubdomainsAliases(domain1, domain2, runtime)
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
		return fmt.Errorf("prepare workflow runtime for urbit clean delete %s: %w", patp, err)
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
	return UrbitCleanDeleteWithRuntime(workflowRuntime{}, patp)
}

func UrbitCleanDeleteWithRuntime(runtime workflowRuntime, patp string) error {
	return urbitCleanDelete(patp, runtime)
}
