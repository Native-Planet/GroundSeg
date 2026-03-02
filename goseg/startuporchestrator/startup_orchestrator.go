package startuporchestrator

import (
	"context"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/config"
	"groundseg/docker/orchestration/subsystem"
	"groundseg/exporter"
	"groundseg/handler/router"
	groundSystem "groundseg/handler/system"
	"groundseg/importer"
	"groundseg/internal/seams"
	"groundseg/leak"
	"groundseg/logger"
	"groundseg/rectify"
	"groundseg/routines"
	"groundseg/routines/logstream"
	"groundseg/session"
	"groundseg/startram"
	"groundseg/startupdeps"
	"groundseg/system"
	"strings"
	"time"

	"go.uber.org/zap"
)

const cloudCheckHost = "1.1.1.1:53"

type startupSubsystemPolicy string

const (
	startupSubsystemRequired startupSubsystemPolicy = "required"
	startupSubsystemOptional startupSubsystemPolicy = "optional"
	startupSubsystemDisabled startupSubsystemPolicy = "disabled"
)

type startupInitCallbackFn func() error

type startupInitCallbackDescriptor struct {
	name        string
	callbackKey string
	policy      startupSubsystemPolicy
	builder     func(startupInitRuntime) startupInitCallbackFn
}

var startupInitSubsystemDescriptors = []startupInitCallbackDescriptor{
	{name: "initialize config subsystem", callbackKey: "initializeConfig", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeConfigFn
	}},
	{name: "initialize auth subsystem", callbackKey: "initializeAuth", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeAuthFn
	}},
	{name: "initialize router subsystem", callbackKey: "initializeRouter", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeRouterFn
	}},
	{name: "initialize system support subsystem", callbackKey: "initializeSystemSupport", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeSystemSupportFn
	}},
	{name: "initialize exporter subsystem", callbackKey: "initializeExporter", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeExporterFn
	}},
	{name: "initialize importer subsystem", callbackKey: "initializeImporter", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeImporterFn
	}},
	{name: "initialize wifi subsystem", callbackKey: "initializeWiFi", policy: startupSubsystemOptional, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeWiFiFn
	}},
	{name: "start mDNS server", callbackKey: "startMDNSServer", policy: startupSubsystemOptional, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.startMDNSServerFn
	}},
	{name: "enable systemd-resolved", callbackKey: "initializeResolved", policy: startupSubsystemOptional, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeResolvedFn
	}},
	{name: "initialize broadcast subsystem", callbackKey: "initializeBroadcast", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeBroadcastFn
	}},
	{name: "initialize docker subsystem", callbackKey: "initializeDocker", policy: startupSubsystemRequired, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.initializeDockerFn
	}},
	{name: "network reachability", callbackKey: "isNetworkReachable", policy: startupSubsystemOptional, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return networkReachabilityCallback(runtime.networkReachabilityFn)
	}},
	{name: "prime rekor key", callbackKey: "primeRekorKey", policy: startupSubsystemOptional, builder: func(runtime startupInitRuntime) startupInitCallbackFn {
		return runtime.primeRekorKeyFn
	}},
}

type startupSubsystemStep struct {
	name   string
	policy startupSubsystemPolicy
	initFn func() error
}

func startupSubsystemRequiredStep(name string, initFn func() error) startupSubsystemStep {
	return startupSubsystemStep{
		name:   name,
		policy: startupSubsystemRequired,
		initFn: initFn,
	}
}

func startupSubsystemOptionalStep(name string, initFn func() error) startupSubsystemStep {
	return startupSubsystemStep{
		name:   name,
		policy: startupSubsystemOptional,
		initFn: initFn,
	}
}

func startupSubsystemAction(name string, policy startupSubsystemPolicy, action func() error) startupSubsystemStep {
	if action == nil {
		return startupSubsystemStep{
			name:   name,
			policy: policy,
		}
	}
	return startupSubsystemStep{
		name:   name,
		policy: policy,
		initFn: action,
	}
}

func runStartupSubsystem(step startupSubsystemStep) error {
	if step.policy == startupSubsystemDisabled {
		return nil
	}
	if step.initFn == nil {
		if step.policy == startupSubsystemRequired {
			return fmt.Errorf("%s initialization callback is not configured", step.name)
		}
		return nil
	}
	if err := step.initFn(); err != nil {
		if step.policy == startupSubsystemOptional {
			zap.L().Warn(fmt.Sprintf("Optional startup subsystem failed: %s: %v", step.name, err))
			return nil
		}
		return fmt.Errorf("%s initialization failed: %w", step.name, err)
	}
	return nil
}

func runStartupSubsystems(steps []startupSubsystemStep) error {
	var startupErrs []error
	for _, step := range steps {
		if err := runStartupSubsystem(step); err != nil {
			startupErrs = append(startupErrs, err)
		}
	}
	if len(startupErrs) == 0 {
		return nil
	}
	return fmt.Errorf("critical startup subsystem initialization failures: %w", errors.Join(startupErrs...))
}

type startupInitRuntime struct {
	initializeConfigFn        startupInitCallbackFn
	initializeAuthFn          startupInitCallbackFn
	initializeRouterFn        startupInitCallbackFn
	initializeSystemSupportFn startupInitCallbackFn
	initializeExporterFn      startupInitCallbackFn
	initializeImporterFn      startupInitCallbackFn
	initializeBroadcastFn     startupInitCallbackFn
	initializeDockerFn        startupInitCallbackFn
	initializeWiFiFn          startupInitCallbackFn
	startMDNSServerFn         startupInitCallbackFn
	initializeResolvedFn      startupInitCallbackFn
	networkReachabilityFn     func(string) bool
	primeRekorKeyFn           startupInitCallbackFn
	ConfigureSwapFn func(string, int) error
	SetupTmpDirFn   func() error
}

func requiredInitCallback(name string, fn func() error) func() error {
	return func() error {
		if fn == nil {
			return fmt.Errorf("%s callback is not configured", name)
		}
		return fn()
	}
}

func networkReachabilityCallback(checkFn func(string) bool) startupInitCallbackFn {
	if checkFn == nil {
		return nil
	}
	return func() error {
		internetAvailable := checkFn(cloudCheckHost)
		zap.L().Info(fmt.Sprintf("Internet available: %t", internetAvailable))
		return nil
	}
}

func startupInitDefaultRuntime() startupInitRuntime {
	return startupInitRuntime{
		initializeConfigFn: func() error { return config.Initialize() },
		initializeAuthFn:   func() error { auth.Initialize(); return nil },
		initializeRouterFn: func() error {
			router.Initialize()
			return nil
		},
		initializeSystemSupportFn: func() error {
			groundSystem.InitializeSupport()
			return nil
		},
		initializeExporterFn:  func() error { return exporter.Initialize() },
		initializeImporterFn:  func() error { return importer.Initialize() },
		initializeBroadcastFn: func() error { return startupdeps.InitializeBroadcast() },
		initializeDockerFn:    func() error { return startupdeps.NewStartupDockerRuntime().Initialize() },
		initializeWiFiFn:      func() error { return system.InitializeWiFi() },
		startMDNSServerFn: func() error {
			routines.StartMDNSServer()
			return nil
		},
		initializeResolvedFn:  func() error { return system.EnableResolved() },
		networkReachabilityFn: config.NetCheck,
		primeRekorKeyFn: func() error { routines.PrimeRekorKey(); return nil },
		ConfigureSwapFn: func(swapFile string, swapVal int) error {
			return system.ConfigureSwap(swapFile, swapVal)
		},
		SetupTmpDirFn: func() error { return system.SetupTmpDir() },
	}
}

func (runtime startupInitRuntime) withDefaults(opts startupInitRuntime) startupInitRuntime {
	if opts.initializeConfigFn != nil {
		runtime.initializeConfigFn = opts.initializeConfigFn
	}
	if opts.initializeAuthFn != nil {
		runtime.initializeAuthFn = opts.initializeAuthFn
	}
	if opts.initializeRouterFn != nil {
		runtime.initializeRouterFn = opts.initializeRouterFn
	}
	if opts.initializeSystemSupportFn != nil {
		runtime.initializeSystemSupportFn = opts.initializeSystemSupportFn
	}
	if opts.initializeExporterFn != nil {
		runtime.initializeExporterFn = opts.initializeExporterFn
	}
	if opts.initializeImporterFn != nil {
		runtime.initializeImporterFn = opts.initializeImporterFn
	}
	if opts.initializeBroadcastFn != nil {
		runtime.initializeBroadcastFn = opts.initializeBroadcastFn
	}
	if opts.initializeResolvedFn != nil {
		runtime.initializeResolvedFn = opts.initializeResolvedFn
	}
	if opts.initializeDockerFn != nil {
		runtime.initializeDockerFn = opts.initializeDockerFn
	}
	if opts.startMDNSServerFn != nil {
		runtime.startMDNSServerFn = opts.startMDNSServerFn
	}
	if opts.initializeWiFiFn != nil {
		runtime.initializeWiFiFn = opts.initializeWiFiFn
	}
	if opts.networkReachabilityFn != nil {
		runtime.networkReachabilityFn = opts.networkReachabilityFn
	}
	if opts.primeRekorKeyFn != nil {
		runtime.primeRekorKeyFn = opts.primeRekorKeyFn
	}
	if opts.ConfigureSwapFn != nil {
		runtime.ConfigureSwapFn = opts.ConfigureSwapFn
	}
	if opts.SetupTmpDirFn != nil {
		runtime.SetupTmpDirFn = opts.SetupTmpDirFn
	}
	return runtime
}

func (runtime startupInitRuntime) initSubsystems() []startupSubsystemStep {
	subsystemSteps := make([]startupSubsystemStep, 0, len(startupInitSubsystemDescriptors))
	for _, descriptor := range startupInitSubsystemDescriptors {
		var stepFn func() error
		switch descriptor.policy {
		case startupSubsystemRequired:
			stepFn = requiredInitCallback(descriptor.name, descriptor.builder(runtime))
		default:
			stepFn = descriptor.builder(runtime)
		}
		subsystemSteps = append(subsystemSteps, startupSubsystemAction(descriptor.name, descriptor.policy, stepFn))
	}
	subsystemSteps = append(subsystemSteps, startupSubsystemOptionalStep("swap configuration", runtime.swapInitializationStep()))
	return subsystemSteps
}

func (runtime startupInitRuntime) validate() []string {
	var missing []string
	for _, descriptor := range startupInitSubsystemDescriptors {
		if descriptor.policy != startupSubsystemRequired {
			continue
		}
		if descriptor.builder(runtime) == nil {
			missing = append(missing, descriptor.callbackKey)
		}
	}
	return missing
}

func (runtime startupInitRuntime) swapInitializationStep() func() error {
	if runtime.ConfigureSwapFn == nil && runtime.SetupTmpDirFn == nil {
		return nil
	}
	return func() error {
		return applySwapAndTmpDirSettings(runtime)
	}
}

type startupBootstrapRuntime struct {
	StartConfigEventLoopFn   func(context.Context) error
	StartStartupContainersFn func(bool)
}

func (runtime StartupRuntime) startupInitSubsystems() []startupSubsystemStep {
	return runtime.startupInitRuntime.initSubsystems()
}

type StartupRuntime struct {
	startupInitRuntime
	startupBootstrapRuntime
}

func defaultStartupRuntime() StartupRuntime {
	return StartupRuntime{
		startupInitRuntime: startupInitDefaultRuntime(),
		startupBootstrapRuntime: startupBootstrapRuntime{
			StartConfigEventLoopFn:   func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel) },
			StartStartupContainersFn: func(bool) {},
		},
	}
}

func (runtime StartupRuntime) withDefaults(opts StartupRuntime) StartupRuntime {
	runtime.startupInitRuntime = runtime.startupInitRuntime.withDefaults(opts.startupInitRuntime)
	runtime.startupBootstrapRuntime = seams.Merge(runtime.startupBootstrapRuntime, opts.startupBootstrapRuntime)
	return runtime
}

type startBackgroundServicesRuntime struct {
	startVersionSubsystemFn     func(context.Context) error
	startDockerSubsystemFn      func(context.Context) error
	startUrbitTransitionFn      func(context.Context) error
	startSystemTransitionFn     func(context.Context) error
	startNewShipTransitionFn    func(context.Context) error
	startImportShipTransitionFn func(context.Context) error
	startRectifyUrbitFn         func(context.Context) error
	syncRetrieveFn              func(context.Context) error
	startLeakFn                    func(context.Context) error
	startSysLogStreamerFn          func(context.Context) error
	startDockerLogStreamerFn       func(context.Context) error
	startDockerLogConnRemoverFn    func(context.Context) error
	startOldLogsCleanerFn          func(context.Context) error
	startDiskUsageWarningFn        func(context.Context) error
	startSmartDiskCheckFn          func(context.Context) error
	startStartramRenewalReminderFn func(context.Context) error
	startPackScheduleLoopFn        func(context.Context) error
	startChopRoutinesFn            func(context.Context) error
	startBackupRoutinesFn          func(context.Context) error
}

func (runtime startBackgroundServicesRuntime) withDefaults(opts startBackgroundServicesRuntime) startBackgroundServicesRuntime {
	if opts.startVersionSubsystemFn != nil {
		runtime.startVersionSubsystemFn = opts.startVersionSubsystemFn
	}
	if opts.startDockerSubsystemFn != nil {
		runtime.startDockerSubsystemFn = opts.startDockerSubsystemFn
	}
	if opts.startUrbitTransitionFn != nil {
		runtime.startUrbitTransitionFn = opts.startUrbitTransitionFn
	}
	if opts.startSystemTransitionFn != nil {
		runtime.startSystemTransitionFn = opts.startSystemTransitionFn
	}
	if opts.startNewShipTransitionFn != nil {
		runtime.startNewShipTransitionFn = opts.startNewShipTransitionFn
	}
	if opts.startImportShipTransitionFn != nil {
		runtime.startImportShipTransitionFn = opts.startImportShipTransitionFn
	}
	if opts.startRectifyUrbitFn != nil {
		runtime.startRectifyUrbitFn = opts.startRectifyUrbitFn
	}
	if opts.syncRetrieveFn != nil {
		runtime.syncRetrieveFn = opts.syncRetrieveFn
	}
	if opts.startLeakFn != nil {
		runtime.startLeakFn = opts.startLeakFn
	}
	if opts.startSysLogStreamerFn != nil {
		runtime.startSysLogStreamerFn = opts.startSysLogStreamerFn
	}
	if opts.startDockerLogStreamerFn != nil {
		runtime.startDockerLogStreamerFn = opts.startDockerLogStreamerFn
	}
	if opts.startDockerLogConnRemoverFn != nil {
		runtime.startDockerLogConnRemoverFn = opts.startDockerLogConnRemoverFn
	}
	if opts.startOldLogsCleanerFn != nil {
		runtime.startOldLogsCleanerFn = opts.startOldLogsCleanerFn
	}
	if opts.startDiskUsageWarningFn != nil {
		runtime.startDiskUsageWarningFn = opts.startDiskUsageWarningFn
	}
	if opts.startSmartDiskCheckFn != nil {
		runtime.startSmartDiskCheckFn = opts.startSmartDiskCheckFn
	}
	if opts.startStartramRenewalReminderFn != nil {
		runtime.startStartramRenewalReminderFn = opts.startStartramRenewalReminderFn
	}
	if opts.startPackScheduleLoopFn != nil {
		runtime.startPackScheduleLoopFn = opts.startPackScheduleLoopFn
	}
	if opts.startChopRoutinesFn != nil {
		runtime.startChopRoutinesFn = opts.startChopRoutinesFn
	}
	if opts.startBackupRoutinesFn != nil {
		runtime.startBackupRoutinesFn = opts.startBackupRoutinesFn
	}
	return runtime
}

func (runtime startBackgroundServicesRuntime) callback(lookup callbackLookupFunc) func(context.Context) error {
	return lookup(runtime)
}

type backgroundServiceDescriptor struct {
	name             string
	callbackLookup   callbackLookupFunc
	requiresCallback bool
	requiresStartram bool
	failFast         bool
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

type callbackLookupFunc func(startBackgroundServicesRuntime) func(context.Context) error

var defaultBackgroundServiceDescriptors = []backgroundServiceDescriptor{
	{name: "version", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startVersionSubsystemFn
	}, requiresCallback: true},
	{name: "docker", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startDockerSubsystemFn
	}, requiresCallback: true},
	{name: "urbit-transition", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startUrbitTransitionFn
	}, requiresCallback: true},
	{name: "system-transition", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startSystemTransitionFn
	}, requiresCallback: true},
	{name: "new-ship-transition", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startNewShipTransitionFn
	}, requiresCallback: true},
	{name: "import-ship-transition", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startImportShipTransitionFn
	}, requiresCallback: true},
	{name: "rectify", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startRectifyUrbitFn
	}, requiresCallback: true},
	{name: "startram-sync", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.syncRetrieveFn
	}, requiresCallback: true, requiresStartram: true, failFast: true},
	{name: "leak", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startLeakFn
	}, requiresCallback: true},
	{name: "sys-log-streamer", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startSysLogStreamerFn
	}, requiresCallback: true},
	{name: "docker-log-streamer", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startDockerLogStreamerFn
	}, requiresCallback: true},
	{name: "docker-log-conn-remover", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startDockerLogConnRemoverFn
	}, requiresCallback: true},
	{name: "old-logs-cleaner", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startOldLogsCleanerFn
	}, requiresCallback: true},
	{name: "disk-usage-warning", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startDiskUsageWarningFn
	}, requiresCallback: true},
	{name: "smart-disk-check", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startSmartDiskCheckFn
	}, requiresCallback: true},
	{name: "startram-renewal", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startStartramRenewalReminderFn
	}, requiresCallback: true, requiresStartram: true},
	{name: "pack-schedule", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startPackScheduleLoopFn
	}, requiresCallback: true},
	{name: "chop-routines", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startChopRoutinesFn
	}, requiresCallback: true},
	{name: "backup-routines", callbackLookup: func(runtime startBackgroundServicesRuntime) func(context.Context) error {
		return runtime.startBackupRoutinesFn
	}, requiresCallback: true},
}

func (runtime startupBootstrapRuntime) validate() []string {
	if runtime.StartStartupContainersFn == nil {
		return []string{"startStartupContainers"}
	}
	return nil
}

func (runtime StartupRuntime) validate() error {
	missing := append([]string{}, runtime.startupInitRuntime.validate()...)
	missing = append(missing, runtime.startupBootstrapRuntime.validate()...)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("startup runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	var missing []string
	for _, service := range defaultBackgroundServiceDescriptors {
		if service.requiresStartram && !startramWgRegistered {
			continue
		}
		if !service.requiresCallback {
			continue
		}
		if runtime.callback(service.callbackLookup) == nil {
			missing = append(missing, service.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

type startupPhase struct {
	id           startupTaskID
	name         string
	dependencies []startupTaskID
	initFn       func() error
	startFn      func(context.Context) error
	healthFn     func(context.Context) error
	required     bool
}

type startupOptions struct {
	httpPort       int
	validateConfig func() error
	startServer    func(context.Context, int) error
	startC2cCheck  func(context.Context) error
	StartupRuntime StartupRuntime
}

const (
	startupTaskCore    startupTaskID = "core-startup"
	startupTaskRuntime startupTaskID = "runtime-startup"
	startupTaskServer  startupTaskID = "service-server"
)

type backgroundServiceHandle struct {
	name string
	stop context.CancelFunc
	err  <-chan error
}

type startupBackgroundServices struct {
	services []backgroundServiceHandle
}

func superviseBackgroundService(ctx context.Context, name string, fn func(context.Context) error) backgroundServiceHandle {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeCtx, stop := context.WithCancel(ctx)
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		defer stop()
		defer func() {
			if recovered := recover(); recovered != nil {
				errCh <- fmt.Errorf("%s background service panic: %v", name, recovered)
			}
		}()
		if err := fn(runtimeCtx); err != nil {
			errCh <- fmt.Errorf("%s background service failed: %w", name, err)
		}
	}()
	return backgroundServiceHandle{name: name, stop: stop, err: errCh}
}

func (services startupBackgroundServices) add(service backgroundServiceHandle) {
	services.services = append(services.services, service)
}

func (services startupBackgroundServices) stop() {
	for _, service := range services.services {
		if service.stop != nil {
			service.stop()
		}
	}
}

func (services startupBackgroundServices) health(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, service := range services.services {
		if service.err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case serviceErr, ok := <-service.err:
			if ok && serviceErr != nil {
				return serviceErr
			}
		default:
		}
	}
	return nil
}

type StartupOptions struct {
	HTTPPort       int
	ValidateConfig func() error
	StartServer    func(context.Context, int) error
	StartC2cCheck  func(context.Context) error
	StartupRuntime StartupRuntime
}

type startupOrchestrator struct {
	tasks []startupPhase
}

func (o startupOrchestrator) init() error {
	orderedTasks, err := o.orderedTasks()
	if err != nil {
		return err
	}
	for _, task := range orderedTasks {
		if task.initFn == nil {
			continue
		}
		if err := task.initFn(); err != nil {
			if task.required {
				return fmt.Errorf("%s init failed: %w", task.name, err)
			}
		}
	}
	return nil
}

func (o startupOrchestrator) start(ctx context.Context) error {
	orderedTasks, err := o.orderedTasks()
	if err != nil {
		return err
	}
	for _, task := range orderedTasks {
		if task.startFn == nil {
			continue
		}
		if err := task.startFn(ctx); err != nil {
			if task.required {
				return fmt.Errorf("%s start failed: %w", task.name, err)
			}
		}
	}
	return nil
}

func (o startupOrchestrator) health(ctx context.Context) error {
	orderedTasks, err := o.orderedTasks()
	if err != nil {
		return err
	}
	for _, task := range orderedTasks {
		if task.healthFn == nil {
			continue
		}
		if err := task.healthFn(ctx); err != nil {
			if task.required {
				return fmt.Errorf("%s health check failed: %w", task.name, err)
			}
		}
	}
	return nil
}

func (o startupOrchestrator) orderedTasks() ([]startupPhase, error) {
	taskLookup := make(map[startupTaskID]startupPhase, len(o.tasks))
	for _, task := range o.tasks {
		if _, exists := taskLookup[task.id]; exists {
			return nil, fmt.Errorf("duplicate startup task: %s", task.id)
		}
		taskLookup[task.id] = task
	}

	state := make(map[startupTaskID]uint8, len(taskLookup))
	ordered := make([]startupPhase, 0, len(taskLookup))

	var visit func(startupTaskID) error
	visit = func(taskID startupTaskID) error {
		status := state[taskID]
		switch status {
		case 1:
			return fmt.Errorf("detected startup task cycle at %s", taskID)
		case 2:
			return nil
		}

		task, ok := taskLookup[taskID]
		if !ok {
			return fmt.Errorf("missing startup task dependency: %s", taskID)
		}

		state[taskID] = 1
		for _, dependency := range task.dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[taskID] = 2
		ordered = append(ordered, task)
		return nil
	}

	for _, task := range o.tasks {
		if err := visit(task.id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func Bootstrap(ctx context.Context, options StartupOptions) error {
	opts := startupOptions{
		httpPort:       options.HTTPPort,
		validateConfig: options.ValidateConfig,
		startServer:    options.StartServer,
		startC2cCheck:  options.StartC2cCheck,
		StartupRuntime: options.StartupRuntime,
	}
	opts.StartupRuntime = defaultStartupRuntime().withDefaults(opts.StartupRuntime)
	if err := opts.StartupRuntime.validate(); err != nil {
		return err
	}
	if opts.httpPort == 0 {
		opts.httpPort = 80
	}
	o := newStartupOrchestrator(opts)
	if err := o.init(); err != nil {
		return err
	}
	if err := o.start(ctx); err != nil {
		return err
	}
	return o.health(ctx)
}

func newStartupOrchestrator(opts startupOptions) startupOrchestrator {
	var backgroundServices *startupBackgroundServices
	tasks := []startupPhase{
		{
			id:       startupTaskCore,
			name:     "core startup",
			required: true,
			initFn: func() error {
				if opts.validateConfig != nil {
					if err := opts.validateConfig(); err != nil {
						return fmt.Errorf("unable to initialize embedded web content: %w", err)
					}
				}
				if err := runStartupSubsystems(opts.StartupRuntime.startupInitSubsystems()); err != nil {
					return err
				}
				return nil
			},
			startFn: func(_ context.Context) error { return nil },
		},
		{
			id:           startupTaskRuntime,
			name:         "runtime startup",
			dependencies: []startupTaskID{startupTaskCore},
			required:     true,
			startFn: func(ctx context.Context) error {
				updateSettings := config.UpdateSettingsSnapshot()
				if opts.StartupRuntime.StartConfigEventLoopFn != nil {
					if err := opts.StartupRuntime.StartConfigEventLoopFn(ctx); err != nil {
						return fmt.Errorf("start config event loop failed: %w", err)
					}
				}
				startramSettings := config.StartramSettingsSnapshot()
				versionUpdateChannel, remoteVersion := startVersionDiscovery(updateSettings.UpdateMode, updateSettings.UpdateBranch)
				services, err := startBackgroundServices(ctx, startramSettings.WgRegistered, opts.startC2cCheck)
				if err != nil {
					return err
				}
				backgroundServices = services

				waitForVersionDiscovery(remoteVersion, versionUpdateChannel, updateSettings.UpdateBranch)
				opts.StartupRuntime.StartStartupContainersFn(startramSettings.WgRegistered)
				return nil
			},
			healthFn: func(ctx context.Context) error {
				if backgroundServices == nil {
					return nil
				}
				return backgroundServices.health(ctx)
			},
		},
		{
			id:           startupTaskServer,
			name:         "service server",
			dependencies: []startupTaskID{startupTaskRuntime},
			required:     true,
			startFn: func(ctx context.Context) error {
				if opts.startServer == nil {
					return fmt.Errorf("startup start server callback is required")
				}
				return opts.startServer(ctx, opts.httpPort)
			},
		},
	}
	return startupOrchestrator{tasks: tasks}
}

func applySwapAndTmpDirSettings(runtimeOps startupInitRuntime) error {
	swapSettings := config.SwapSettingsSnapshot()
	var startupErrs []error
	zap.L().Info(fmt.Sprintf("Setting up swap %v for %vG", swapSettings.SwapFile, swapSettings.SwapVal))
	if runtimeOps.ConfigureSwapFn == nil {
		return nil
	}
	if err := runtimeOps.ConfigureSwapFn(swapSettings.SwapFile, swapSettings.SwapVal); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
		startupErrs = append(startupErrs, fmt.Errorf("unable to set swap: %w", err))
	}
	zap.L().Info("Setting up /tmp directory")
	if runtimeOps.SetupTmpDirFn == nil {
		if len(startupErrs) == 0 {
			return nil
		}
		return fmt.Errorf("unable to setup swap or /tmp: %w", errors.Join(startupErrs...))
	}
	if err := runtimeOps.SetupTmpDirFn(); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to setup /tmp: %v", err))
		startupErrs = append(startupErrs, fmt.Errorf("unable to setup /tmp: %w", err))
	}
	if len(startupErrs) == 0 {
		return nil
	}
	return fmt.Errorf("unable to setup swap or /tmp: %w", errors.Join(startupErrs...))
}

func startVersionDiscovery(updateMode, releaseChannel string) (chan bool, bool) {
	versionUpdateChannel := make(chan bool, 1)
	if updateMode == "auto" {
		go func() {
			_, versionUpdate := config.SyncVersionInfo()
			versionUpdateChannel <- versionUpdate
		}()
		return versionUpdateChannel, true
	}
	versionStruct := config.LocalVersion()
	config.SetVersionChannel(versionStruct.Groundseg[releaseChannel])
	return versionUpdateChannel, false
}

func waitForVersionDiscovery(remoteVersion bool, versionUpdate <-chan bool, updateBranch string) {
	if !remoteVersion {
		return
	}
	select {
	case versionUpdateSuccess := <-versionUpdate:
		if versionUpdateSuccess {
			zap.L().Info("Version info retrieved")
			return
		}
		zap.L().Warn("Could not retrieve version info from remote. Falling back to local version metadata.")
	case <-time.After(10 * time.Second):
		zap.L().Warn("Could not retrieve version info after 10 seconds!")
	}
	versionStruct := config.LocalVersion()
	versionFromBranch, ok := versionStruct.Groundseg[updateBranch]
	if !ok {
		zap.L().Warn(fmt.Sprintf("Could not resolve local version channel %q for fallback", updateBranch))
		return
	}
	config.SetVersionChannel(versionFromBranch)
}

func startBackgroundServices(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context) error) (*startupBackgroundServices, error) {
	return startBackgroundServicesWithRuntime(ctx, startramWgRegistered, startC2cCheck, defaultStartBackgroundServicesRuntime())
}

func startBackgroundServicesWithRuntime(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context) error, runtime startBackgroundServicesRuntime) (*startupBackgroundServices, error) {
	runtime = defaultStartBackgroundServicesRuntime().withDefaults(runtime)
	if err := runtime.validate(startramWgRegistered); err != nil {
		return nil, err
	}
	systemRuntime := session.LogstreamRuntimeState()
	logger.ConfigureLogstreamRuntime(systemRuntime)
	logstream.Configure(systemRuntime, systemRuntime.SystemLogMessages())
	services := &startupBackgroundServices{}
	if startC2cCheck != nil {
		services.add(superviseBackgroundService(ctx, "c2c-check", func(ctx context.Context) error {
			return startC2cCheck(ctx)
		}))
	}
	for _, descriptor := range defaultBackgroundServiceDescriptors {
		if descriptor.requiresStartram && !startramWgRegistered {
			continue
		}
		startFn := runtime.callback(descriptor.callbackLookup)
		if startFn == nil {
			missing := descriptor.name
			return nil, fmt.Errorf("start background services runtime missing required callback: %s", missing)
		}
		handle := superviseBackgroundService(ctx, descriptor.name, startFn)
		services.add(handle)
		if !descriptor.failFast {
			continue
		}
		select {
		case serviceErr, ok := <-handle.err:
			if ok && serviceErr != nil {
				return nil, serviceErr
			}
		default:
		}
	}
	return services, nil
}

func defaultStartBackgroundServicesRuntime() startBackgroundServicesRuntime {
	return startBackgroundServicesRuntime{
		startVersionSubsystemFn:     routines.StartVersionSubsystemWithContext,
		startDockerSubsystemFn:      subsystem.StartDockerSubsystemWithContext,
		startUrbitTransitionFn:      rectify.UrbitTransitionHandlerWithContext,
		startSystemTransitionFn:     rectify.SystemTransitionHandlerWithContext,
		startNewShipTransitionFn:    rectify.NewShipTransitionHandlerWithContext,
		startImportShipTransitionFn: rectify.ImportShipTransitionHandlerWithContext,
		startRectifyUrbitFn:         rectify.RectifyUrbitWithContext,
		syncRetrieveFn: func(context.Context) error {
			return syncRetrieveWithStartram()
		},
		startLeakFn:                    leak.StartLeakWithContext,
		startSysLogStreamerFn:          logstream.SysLogStreamerWithContext,
		startDockerLogStreamerFn:       logstream.DockerLogStreamerWithContext,
		startDockerLogConnRemoverFn:    logstream.DockerLogConnRemoverWithContext,
		startOldLogsCleanerFn:          logstream.OldLogsCleanerWithContext,
		startDiskUsageWarningFn:        routines.DiskUsageWarningWithContext,
		startSmartDiskCheckFn:          routines.SmartDiskCheckWithContext,
		startStartramRenewalReminderFn: routines.StartramRenewalReminderWithContext,
		startPackScheduleLoopFn:        routines.PackScheduleLoopWithContext,
		startChopRoutinesFn:            routines.StartChopRoutinesWithContext,
		startBackupRoutinesFn:          routines.StartBackupRoutinesWithContext,
	}
}

func loadService(loadFn func() error, failureMessage string) error {
	if loadFn == nil {
		return fmt.Errorf("startup load function is not configured: %s", failureMessage)
	}
	if err := loadFn(); err != nil {
		zap.L().Error(fmt.Sprintf("%s: %v", failureMessage, err))
		return err
	}
	return nil
}
