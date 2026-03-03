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
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

type workflowRuntime struct {
	dockerOrchestration.DockerTransitionRuntime
	dockerOrchestration.DockerHealthRuntime
	EmitTransitionFn     func(patp string, transitionType string, event string)
	SleepFn              func(time.Duration)
	DispatchUploadImport func(context.Context, UploadImportCommand) error
	LookupCNAMEFn        func(string) (string, error)
	BarExitFn            func(string) error
	StopContainerFn      func(string) error
	DeleteContainerFn    func(string) error
}

var errUploadImportCoordinatorUnconfigured = fmt.Errorf("workflow runtime upload import coordinator is not configured")
var errShipStatusNotFound = errors.New("ship status doesn't exist")

type workflowAliasLookupFn func(string) (string, error)

type transitionPlan[E comparable] = transitionlifecycle.LifecyclePlan[E]
type transitionStep[E comparable] = transitionlifecycle.LifecycleStep[E]

func runTransitionLifecycle[E comparable](runtime workflowRuntime, emit func(E), plan transitionPlan[E], steps ...transitionStep[E]) error {
	runtime = withDefaultsWorkflowRuntime(runtime)
	return transitionlifecycle.RunLifecycle(
		transitionlifecycle.Runtime[E]{
			Emit:  emit,
			Sleep: runtime.SleepFn,
		},
		plan,
		steps...,
	)
}

func runUrbitTransition(patp string, transitionType string, plan transitionPlan[string], steps ...transitionStep[string]) error {
	runtime := defaultWorkflowRuntime()
	return runTransitionLifecycle[string](
		runtime,
		func(event string) {
			emitUrbitTransition(runtime, patp, transitionType, event)
		},
		plan,
		steps...,
	)
}

func newWorkflowRuntime() workflowRuntime {
	defaultEmit := func(patp, transitionType, event string) {
		events.DefaultEventRuntime().PublishUrbitTransition(context.Background(), structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
	}
	defaultDispatch := func(ctx context.Context, cmd UploadImportCommand) error {
		return unconfiguredUploadImportCoordinator(ctx, cmd)
	}
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return workflowRuntime{
		DockerTransitionRuntime: dockerOrchestration.NewDockerTransitionRuntime(orchestrationRuntime),
		DockerHealthRuntime:     dockerOrchestration.NewDockerHealthRuntime(orchestrationRuntime),
		EmitTransitionFn:        defaultEmit,
		SleepFn:                 time.Sleep,
		DispatchUploadImport:    defaultDispatch,
		LookupCNAMEFn:           net.LookupCNAME,
		BarExitFn:               click.BarExit,
		StopContainerFn:         orchestrationRuntime.StopContainerByNameFn,
		DeleteContainerFn:       orchestrationRuntime.DeleteContainerFn,
	}
}

var defaultWorkflowRuntimeValue = newWorkflowRuntime()

func defaultWorkflowRuntime() workflowRuntime {
	return defaultWorkflowRuntimeValue
}

func resolveWorkflowRuntime(overrides ...workflowRuntime) workflowRuntime {
	if len(overrides) == 0 {
		return defaultWorkflowRuntime()
	}
	return withDefaultsWorkflowRuntime(overrides[0])
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
		runtime.DispatchUploadImport = coordinator
	}
	return dispatchUploadImport(runtime, ctx, cmd)
}

func dispatchUploadImport(runtime workflowRuntime, ctx context.Context, cmd UploadImportCommand) error {
	runtime = withDefaultsWorkflowRuntime(runtime)
	return runtime.DispatchUploadImport(ctx, cmd)
}

func emitUrbitTransition(runtime workflowRuntime, patp, transitionType, event string) {
	if runtime.EmitTransitionFn == nil {
		return
	}
	runtime.EmitTransitionFn(patp, transitionType, event)
}

func PublishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) {
	publishTransition(defaultWorkflowRuntime(), publish, event, clear, clearDelay)
}

func publishTransition[T any](runtime workflowRuntime, publish func(T), event T, clear T, clearDelay time.Duration) {
	runtime = withDefaultsWorkflowRuntime(runtime)
	publish(event)
	if clearDelay > 0 {
		runtime.SleepFn(clearDelay)
	}
	publish(clear)
}

func RunTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	runtime := defaultWorkflowRuntime()
	return runTransitionLifecycle[string](
		runtime,
		func(event string) {
			emitUrbitTransition(runtime, patp, transitionType, event)
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
	return runTransitionLifecycle[string](
		runtime,
		func(event string) {
			emitUrbitTransition(runtime, patp, transitionType, event)
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
	runtime := resolveWorkflowRuntime(defaultWorkflowRuntime())
	phaseSteps := make([]transitionStep[string], 0, len(steps))
	for _, step := range steps {
		phaseSteps = append(phaseSteps, transitionStep[string]{
			Event: string(step.Phase),
			Run:   step.Run,
		})
	}

	return runTransitionLifecycle[string](
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
	runtime := defaultWorkflowRuntime()
	runtime = withDefaultsWorkflowRuntime(runtime)
	return WaitForUrbitStop(patp, func(piers []string) (map[string]string, error) {
		return runtime.GetShipStatusFn(piers)
	}, PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error, runtime ...workflowRuntime) error {
	resolvedRuntime := resolveWorkflowRuntime(runtime...)
	resolvedRuntime = withDefaultsWorkflowRuntime(resolvedRuntime)
	return PersistUrbitConfig(patp, mutate, resolvedRuntime.UpdateUrbitFn)
}

func persistShipUrbitConfig[T any](patp string, section dockerOrchestration.UrbitConfigSection, mutate func(*T) error, runtime ...workflowRuntime) error {
	resolvedRuntime := resolveWorkflowRuntime(runtime...)
	resolvedRuntime = withDefaultsWorkflowRuntime(resolvedRuntime)
	switch section {
	case dockerOrchestration.UrbitConfigSectionRuntime:
		runtimeMutate, ok := any(mutate).(func(*structs.UrbitRuntimeConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitRuntimeConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitRuntimeConfig(patp, runtimeMutate)
	case dockerOrchestration.UrbitConfigSectionNetwork:
		networkMutate, ok := any(mutate).(func(*structs.UrbitNetworkConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitNetworkConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitNetworkConfig(patp, networkMutate)
	case dockerOrchestration.UrbitConfigSectionSchedule:
		scheduleMutate, ok := any(mutate).(func(*structs.UrbitScheduleConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitScheduleConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitScheduleConfig(patp, scheduleMutate)
	case dockerOrchestration.UrbitConfigSectionFeature:
		featureMutate, ok := any(mutate).(func(*structs.UrbitFeatureConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitFeatureConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitFeatureConfig(patp, featureMutate)
	case dockerOrchestration.UrbitConfigSectionWeb:
		webMutate, ok := any(mutate).(func(*structs.UrbitWebConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitWebConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitWebConfig(patp, webMutate)
	case dockerOrchestration.UrbitConfigSectionBackup:
		backupMutate, ok := any(mutate).(func(*structs.UrbitBackupConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitBackupConfig)", section)
		}
		return resolvedRuntime.UpdateUrbitBackupConfig(patp, backupMutate)
	default:
		return fmt.Errorf("unsupported urbit config section: %s", section)
	}
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate)
}

func areSubdomainsAliases(domain1, domain2 string, runtime ...workflowRuntime) (bool, error) {
	resolvedRuntime := resolveWorkflowRuntime(runtime...)
	resolvedRuntime = withDefaultsWorkflowRuntime(resolvedRuntime)
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("Invalid subdomain")
	}
	if config.GetStartramConfig().Cname != "" && domain1[firstDot+1:] == config.GetStartramConfig().Cname {
		return true, nil
	}
	cname1, err := resolvedRuntime.LookupCNAMEFn(domain1)
	if err != nil {
		return false, fmt.Errorf("lookup CNAME for %s: %w", domain1, err)
	}
	cname2, err := resolvedRuntime.LookupCNAMEFn(domain2)
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
		LookupCNAMEFn: workflowAliasLookupFn(lookupAlias),
	}
	return areSubdomainsAliases(domain1, domain2, runtime)
}

func urbitCleanDelete(patp string, runtime ...workflowRuntime) error {
	resolvedRuntime := resolveWorkflowRuntime(runtime...)
	resolvedRuntime = withDefaultsWorkflowRuntime(resolvedRuntime)
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := resolvedRuntime.GetShipStatusFn([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
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
			if err := resolvedRuntime.BarExitFn(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit: %v", patp, err))
				if err = resolvedRuntime.StopContainerFn(patp); err != nil {
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
			resolvedRuntime.SleepFn(1 * time.Second)
		}
	}
	return resolvedRuntime.DeleteContainerFn(patp)
}

func withDefaultsWorkflowRuntime(runtime workflowRuntime) workflowRuntime {
	base := defaultWorkflowRuntime()
	return seams.Merge(base, runtime)
}

func shipStatusNotFoundErr(patp string) error {
	return fmt.Errorf("%w: %s", errShipStatusNotFound, patp)
}

func UrbitCleanDelete(patp string) error {
	return urbitCleanDelete(patp)
}
