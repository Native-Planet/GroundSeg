package shipworkflow

import (
	"context"
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/events"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/lifecycle"
	"groundseg/orchestration"
	"groundseg/structs"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

type workflowRuntime struct {
	emitTransitionFn       func(patp string, transitionType string, event string)
	sleepFn                func(time.Duration)
	dispatchUploadImportFn func(context.Context, UploadImportCommand) error
	lookupCNAMEFn          func(string) (string, error)
	getShipStatusFn        func([]string) (map[string]string, error)
	barExitFn              func(string) error
	stopContainerFn        func(string) error
	deleteContainerFn      func(string) error
}

var sleepForWorkflow = time.Sleep
var errUploadImportCoordinatorUnconfigured = errors.New("workflow runtime upload import coordinator is not configured")
var persistUrbitConfigFn = config.UpdateUrbit

type workflowAliasLookupFn func(string) (string, error)

type transitionPlan[E comparable] struct {
	EmitStart    bool
	StartEvent   E
	SuccessEvent E
	EmitSuccess  bool
	ErrorEvent   func(error) E
	ClearEvent   E
	ClearDelay   time.Duration
}

type transitionStep[E comparable] struct {
	Event    E
	EmitWhen func() bool
	Run      func() error
}

func runTransitionLifecycleWithRuntime[E comparable](runtime workflowRuntime, emit func(E), plan transitionPlan[E], steps ...transitionStep[E]) error {
	runtime = withDefaultsWorkflowRuntime(runtime)

	if plan.EmitStart {
		emit(plan.StartEvent)
	}

	var zero E
	policy := orchestration.NewTransitionPolicy(plan.ClearDelay, func(d time.Duration) {
		runtime.sleepFn(d)
	})
	defer policy.Cleanup(func() {
		emit(plan.ClearEvent)
	})

	for _, step := range steps {
		if step.Event != zero && (step.EmitWhen == nil || step.EmitWhen()) {
			emit(step.Event)
		}
		if step.Run == nil {
			continue
		}
		if err := step.Run(); err != nil {
			emit(plan.ErrorEvent(err))
			return err
		}
	}

	if plan.EmitSuccess {
		emit(plan.SuccessEvent)
	}

	return nil
}

func runUrbitTransitionWithRuntime(runtime workflowRuntime, patp string, transitionType string, plan transitionPlan[string], steps ...transitionStep[string]) error {
	return runTransitionLifecycleWithRuntime[string](
		runtime,
		func(event string) {
			emitUrbitTransition(runtime, patp, transitionType, event)
		},
		plan,
		steps...,
	)
}

func runUrbitTransition(patp string, transitionType string, plan transitionPlan[string], steps ...transitionStep[string]) error {
	return runUrbitTransitionWithRuntime(defaultWorkflowRuntime(), patp, transitionType, plan, steps...)
}

func defaultWorkflowRuntime() workflowRuntime {
	defaultEmit := func(patp, transitionType, event string) {
		events.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
	}
	defaultDispatch := func(ctx context.Context, cmd UploadImportCommand) error {
		return unconfiguredUploadImportCoordinator(ctx, cmd)
	}
	return workflowRuntime{
		emitTransitionFn:       defaultEmit,
		sleepFn:                sleepForWorkflow,
		dispatchUploadImportFn: defaultDispatch,
		lookupCNAMEFn:          net.LookupCNAME,
		getShipStatusFn:        dockerOrchestration.GetShipStatus,
		barExitFn:              click.BarExit,
		stopContainerFn:        dockerOrchestration.StopContainerByName,
		deleteContainerFn:      dockerOrchestration.DeleteContainer,
	}
}

type UploadImportCommand struct {
	ArchivePath string
	Filename    string
	Patp        string
	Remote      bool
	Fix         bool
	CustomDrive string
}

type UploadImportCoordinator interface {
	HandleUploadImport(context.Context, UploadImportCommand) error
}

type UploadImportCoordinatorFunc func(context.Context, UploadImportCommand) error

func (f UploadImportCoordinatorFunc) HandleUploadImport(ctx context.Context, cmd UploadImportCommand) error {
	return f(ctx, cmd)
}

func unconfiguredUploadImportCoordinator(context.Context, UploadImportCommand) error {
	return errUploadImportCoordinatorUnconfigured
}

func DispatchUploadImport(ctx context.Context, cmd UploadImportCommand) error {
	return DispatchUploadImportWithCoordinator(nil, ctx, cmd)
}

func DispatchUploadImportWithCoordinator(coordinator UploadImportCoordinator, ctx context.Context, cmd UploadImportCommand) error {
	runtime := defaultWorkflowRuntime()
	if coordinator != nil {
		runtime.dispatchUploadImportFn = coordinator.HandleUploadImport
	}
	return dispatchUploadImport(runtime, ctx, cmd)
}

func dispatchUploadImport(runtime workflowRuntime, ctx context.Context, cmd UploadImportCommand) error {
	runtime = withDefaultsWorkflowRuntime(runtime)
	return runtime.dispatchUploadImportFn(ctx, cmd)
}

func emitUrbitTransition(runtime workflowRuntime, patp, transitionType, event string) {
	if runtime.emitTransitionFn == nil {
		return
	}
	runtime.emitTransitionFn(patp, transitionType, event)
}

func PublishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) {
	publishTransition(defaultWorkflowRuntime(), publish, event, clear, clearDelay)
}

func publishTransition[T any](runtime workflowRuntime, publish func(T), event T, clear T, clearDelay time.Duration) {
	runtime = withDefaultsWorkflowRuntime(runtime)
	publish(event)
	if clearDelay > 0 {
		runtime.sleepFn(clearDelay)
	}
	publish(clear)
}

func RunTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	return runTransitionedOperationWithRuntime(
		defaultWorkflowRuntime(),
		patp,
		transitionType,
		startEvent,
		successEvent,
		clearDelay,
		operation,
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
	return runUrbitTransitionWithRuntime(
		runtime,
		patp,
		transitionType,
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
	return runPhaseWorkflowWithRuntime(
		defaultWorkflowRuntime(),
		patp,
		transitionType,
		successEvent,
		clearDelay,
		steps...,
	)
}

func runPhaseWorkflowWithRuntime(
	runtime workflowRuntime,
	patp string,
	transitionType string,
	successEvent string,
	clearDelay time.Duration,
	steps ...lifecycle.Step,
) error {
	phaseSteps := make([]transitionStep[string], 0, len(steps))
	for _, step := range steps {
		phaseSteps = append(phaseSteps, transitionStep[string]{
			Event: string(step.Phase),
			Run:   step.Run,
		})
	}

	return runTransitionLifecycleWithRuntime[string](
		runtime,
		func(event string) {
			emitUrbitTransition(runtime, patp, transitionType, event)
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
	return waitCompleteWithRuntime(defaultWorkflowRuntime(), patp)
}

func waitCompleteWithRuntime(runtime workflowRuntime, patp string) error {
	runtime = withDefaultsWorkflowRuntime(runtime)
	return WaitForUrbitStop(patp, func(piers []string) (map[string]string, error) {
		return runtime.getShipStatusFn(piers)
	}, PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error) error {
	return PersistUrbitConfig(patp, mutate, persistUrbitConfigFn)
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate)
}

func areSubdomainsAliases(domain1, domain2 string) (bool, error) {
	return areSubdomainsAliasesWithRuntime(defaultWorkflowRuntime(), domain1, domain2)
}

func areSubdomainsAliasesWithRuntime(runtime workflowRuntime, domain1, domain2 string) (bool, error) {
	runtime = withDefaultsWorkflowRuntime(runtime)
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("Invalid subdomain")
	}
	if config.GetStartramConfig().Cname != "" && domain1[firstDot+1:] == config.GetStartramConfig().Cname {
		return true, nil
	}
	cname1, err := runtime.lookupCNAMEFn(domain1)
	if err != nil {
		return false, err
	}
	cname2, err := runtime.lookupCNAMEFn(domain2)
	if err != nil {
		return false, err
	}
	return cname1 == cname2, nil
}

func AreSubdomainsAliases(domain1, domain2 string) (bool, error) {
	return areSubdomainsAliasesWithRuntime(defaultWorkflowRuntime(), domain1, domain2)
}

func AreSubdomainsAliasesWithLookup(
	lookupAlias func(string) (string, error),
	domain1 string,
	domain2 string,
) (bool, error) {
	runtime := workflowRuntime{
		lookupCNAMEFn: workflowAliasLookupFn(lookupAlias),
	}
	return areSubdomainsAliasesWithRuntime(runtime, domain1, domain2)
}

func urbitCleanDelete(patp string) error {
	return urbitCleanDeleteWithRuntime(defaultWorkflowRuntime(), patp)
}

func urbitCleanDeleteWithRuntime(runtime workflowRuntime, patp string) error {
	runtime = withDefaultsWorkflowRuntime(runtime)
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := runtime.getShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	status, err := getShipRunningStatus(patp)
	if err == nil {
		if strings.Contains(status, "Up") {
			if err := runtime.barExitFn(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit: %v", patp, err))
				if err = runtime.stopContainerFn(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop %s: %v", patp, err))
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
			runtime.sleepFn(1 * time.Second)
		}
	}
	return runtime.deleteContainerFn(patp)
}

func withDefaultsWorkflowRuntime(runtime workflowRuntime) workflowRuntime {
	returned := defaultWorkflowRuntime()
	setIfNonNil(&returned.emitTransitionFn, runtime.emitTransitionFn)
	setIfNonNil(&returned.sleepFn, runtime.sleepFn)
	setIfNonNil(&returned.dispatchUploadImportFn, runtime.dispatchUploadImportFn)
	setIfNonNil(&returned.lookupCNAMEFn, runtime.lookupCNAMEFn)
	setIfNonNil(&returned.getShipStatusFn, runtime.getShipStatusFn)
	setIfNonNil(&returned.barExitFn, runtime.barExitFn)
	setIfNonNil(&returned.stopContainerFn, runtime.stopContainerFn)
	setIfNonNil(&returned.deleteContainerFn, runtime.deleteContainerFn)
	return returned
}

func setIfNonNil[T any](target *T, value T) {
	if any(value) == nil {
		return
	}
	*target = value
}

func UrbitCleanDelete(patp string) error {
	return urbitCleanDelete(patp)
}
