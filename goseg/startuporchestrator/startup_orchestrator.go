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
}

const (
	callbackKeyInitializeConfig        = "initializeConfig"
	callbackKeyInitializeAuth         = "initializeAuth"
	callbackKeyInitializeRouter       = "initializeRouter"
	callbackKeyInitializeSystemSupport = "initializeSystemSupport"
	callbackKeyInitializeExporter     = "initializeExporter"
	callbackKeyInitializeImporter     = "initializeImporter"
	callbackKeyInitializeWiFi          = "initializeWiFi"
	callbackKeyStartMDNSServer         = "startMDNSServer"
	callbackKeyInitializeResolved      = "initializeResolved"
	callbackKeyInitializeBroadcast     = "initializeBroadcast"
	callbackKeyInitializeDocker        = "initializeDocker"
	callbackKeyNetworkReachability     = "isNetworkReachable"
	callbackKeyPrimeRekorKey          = "primeRekorKey"
)

func (runtime startupInitRuntime) callback(key string) startupInitCallbackFn {
	switch key {
	case callbackKeyInitializeConfig:
		return runtime.initializeConfigFn
	case callbackKeyInitializeAuth:
		return runtime.initializeAuthFn
	case callbackKeyInitializeRouter:
		return runtime.initializeRouterFn
	case callbackKeyInitializeSystemSupport:
		return runtime.initializeSystemSupportFn
	case callbackKeyInitializeExporter:
		return runtime.initializeExporterFn
	case callbackKeyInitializeImporter:
		return runtime.initializeImporterFn
	case callbackKeyInitializeWiFi:
		return runtime.initializeWiFiFn
	case callbackKeyStartMDNSServer:
		return runtime.startMDNSServerFn
	case callbackKeyInitializeResolved:
		return runtime.initializeResolvedFn
	case callbackKeyInitializeBroadcast:
		return runtime.initializeBroadcastFn
	case callbackKeyInitializeDocker:
		return runtime.initializeDockerFn
	case callbackKeyNetworkReachability:
		return networkReachabilityCallback(runtime.networkReachabilityFn)
	case callbackKeyPrimeRekorKey:
		return runtime.primeRekorKeyFn
	default:
		return nil
	}
}

var startupInitSubsystemDescriptors = []startupInitCallbackDescriptor{
	{name: "initialize config subsystem", callbackKey: callbackKeyInitializeConfig, policy: startupSubsystemRequired},
	{name: "initialize auth subsystem", callbackKey: callbackKeyInitializeAuth, policy: startupSubsystemRequired},
	{name: "initialize router subsystem", callbackKey: callbackKeyInitializeRouter, policy: startupSubsystemRequired},
	{name: "initialize system support subsystem", callbackKey: callbackKeyInitializeSystemSupport, policy: startupSubsystemRequired},
	{name: "initialize exporter subsystem", callbackKey: callbackKeyInitializeExporter, policy: startupSubsystemRequired},
	{name: "initialize importer subsystem", callbackKey: callbackKeyInitializeImporter, policy: startupSubsystemRequired},
	{name: "initialize wifi subsystem", callbackKey: callbackKeyInitializeWiFi, policy: startupSubsystemOptional},
	{name: "start mDNS server", callbackKey: callbackKeyStartMDNSServer, policy: startupSubsystemOptional},
	{name: "enable systemd-resolved", callbackKey: callbackKeyInitializeResolved, policy: startupSubsystemOptional},
	{name: "initialize broadcast subsystem", callbackKey: callbackKeyInitializeBroadcast, policy: startupSubsystemRequired},
	{name: "initialize docker subsystem", callbackKey: callbackKeyInitializeDocker, policy: startupSubsystemRequired},
	{name: "network reachability", callbackKey: callbackKeyNetworkReachability, policy: startupSubsystemOptional},
	{name: "prime rekor key", callbackKey: callbackKeyPrimeRekorKey, policy: startupSubsystemOptional},
}

type startupSubsystemStep struct {
	name   string
	policy startupSubsystemPolicy
	initFn func() error
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
	ConfigureSwapFn           func(string, int) error
	SetupTmpDirFn             func() error
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
		primeRekorKeyFn:       func() error { routines.PrimeRekorKey(); return nil },
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
		subsystemSteps = append(subsystemSteps, startupSubsystemStep{
			name:   descriptor.name,
			policy: descriptor.policy,
			initFn: runtime.callback(descriptor.callbackKey),
		})
	}
	if runtime.swapInitializationStep() != nil {
		subsystemSteps = append(subsystemSteps, startupSubsystemStep{
			name:   "swap configuration",
			policy: startupSubsystemOptional,
			initFn: runtime.swapInitializationStep(),
		})
	}
	return subsystemSteps
}

func (runtime startupInitRuntime) validate() []string {
	var missing []string
	for _, descriptor := range startupInitSubsystemDescriptors {
		if descriptor.policy != startupSubsystemRequired {
			continue
		}
		if runtime.callback(descriptor.callbackKey) == nil {
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
			StartConfigEventLoopFn:   func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel()) },
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
	startVersionSubsystemFn        func(context.Context) error
	startDockerSubsystemFn         func(context.Context) error
	startUrbitTransitionFn         func(context.Context) error
	startSystemTransitionFn        func(context.Context) error
	startNewShipTransitionFn       func(context.Context) error
	startImportShipTransitionFn    func(context.Context) error
	startRectifyUrbitFn            func(context.Context) error
	syncRetrieveFn                 func(context.Context) error
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

func (runtime startBackgroundServicesRuntime) callback(key string) func(context.Context) error {
	switch key {
	case "version":
		return runtime.startVersionSubsystemFn
	case "docker":
		return runtime.startDockerSubsystemFn
	case "urbit-transition":
		return runtime.startUrbitTransitionFn
	case "system-transition":
		return runtime.startSystemTransitionFn
	case "new-ship-transition":
		return runtime.startNewShipTransitionFn
	case "import-ship-transition":
		return runtime.startImportShipTransitionFn
	case "rectify":
		return runtime.startRectifyUrbitFn
	case "startram-sync":
		return runtime.syncRetrieveFn
	case "leak":
		return runtime.startLeakFn
	case "sys-log-streamer":
		return runtime.startSysLogStreamerFn
	case "docker-log-streamer":
		return runtime.startDockerLogStreamerFn
	case "docker-log-conn-remover":
		return runtime.startDockerLogConnRemoverFn
	case "old-logs-cleaner":
		return runtime.startOldLogsCleanerFn
	case "disk-usage-warning":
		return runtime.startDiskUsageWarningFn
	case "smart-disk-check":
		return runtime.startSmartDiskCheckFn
	case "startram-renewal":
		return runtime.startStartramRenewalReminderFn
	case "pack-schedule":
		return runtime.startPackScheduleLoopFn
	case "chop-routines":
		return runtime.startChopRoutinesFn
	case "backup-routines":
		return runtime.startBackupRoutinesFn
	default:
		return nil
	}
}

type backgroundServiceDescriptor struct {
	name             string
	callbackKey      string
	requiresCallback bool
	requiresStartram bool
	failFast         bool
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

var defaultBackgroundServiceDescriptors = []backgroundServiceDescriptor{
	{name: "version", callbackKey: "version", requiresCallback: true},
	{name: "docker", callbackKey: "docker", requiresCallback: true},
	{name: "urbit-transition", callbackKey: "urbit-transition", requiresCallback: true},
	{name: "system-transition", callbackKey: "system-transition", requiresCallback: true},
	{name: "new-ship-transition", callbackKey: "new-ship-transition", requiresCallback: true},
	{name: "import-ship-transition", callbackKey: "import-ship-transition", requiresCallback: true},
	{name: "rectify", callbackKey: "rectify", requiresCallback: true},
	{name: "startram-sync", callbackKey: "startram-sync", requiresCallback: true, requiresStartram: true, failFast: true},
	{name: "leak", callbackKey: "leak", requiresCallback: true},
	{name: "sys-log-streamer", callbackKey: "sys-log-streamer", requiresCallback: true},
	{name: "docker-log-streamer", callbackKey: "docker-log-streamer", requiresCallback: true},
	{name: "docker-log-conn-remover", callbackKey: "docker-log-conn-remover", requiresCallback: true},
	{name: "old-logs-cleaner", callbackKey: "old-logs-cleaner", requiresCallback: true},
	{name: "disk-usage-warning", callbackKey: "disk-usage-warning", requiresCallback: true},
	{name: "smart-disk-check", callbackKey: "smart-disk-check", requiresCallback: true},
	{name: "startram-renewal", callbackKey: "startram-renewal", requiresCallback: true, requiresStartram: true},
	{name: "pack-schedule", callbackKey: "pack-schedule", requiresCallback: true},
	{name: "chop-routines", callbackKey: "chop-routines", requiresCallback: true},
	{name: "backup-routines", callbackKey: "backup-routines", requiresCallback: true},
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
		if runtime.callback(service.callbackKey) == nil {
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
		startFn := runtime.callback(descriptor.callbackKey)
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
		startDockerSubsystemFn:      subsystem.StartDockerSubsystem,
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
