package main

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
	"groundseg/leak"
	"groundseg/rectify"
	"groundseg/routines"
	"groundseg/startram"
	"groundseg/startupdeps"
	"groundseg/system"
	"strings"
	"time"

	"go.uber.org/zap"
)

type startupSubsystemPolicy string

const (
	startupSubsystemRequired startupSubsystemPolicy = "required"
	startupSubsystemOptional startupSubsystemPolicy = "optional"
	startupSubsystemDisabled startupSubsystemPolicy = "disabled"
)

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

func (runtime startupRuntime) startupInitSubsystems() []startupSubsystemStep {
	return []startupSubsystemStep{
		startupSubsystemRequiredStep("initialize config subsystem", runtime.initializeConfigFn),
		startupSubsystemAction("initialize auth subsystem", startupSubsystemRequired, runtime.initializeAuthFn),
		startupSubsystemAction("initialize router subsystem", startupSubsystemRequired, runtime.initializeRouterFn),
		startupSubsystemAction("initialize system support subsystem", startupSubsystemRequired, runtime.initializeSystemSupportFn),
		startupSubsystemRequiredStep("initialize exporter subsystem", runtime.initializeExporterFn),
		startupSubsystemRequiredStep("initialize importer subsystem", runtime.initializeImporterFn),
		startupSubsystemOptionalStep("initialize wifi subsystem", runtime.initializeWiFiFn),
		startupSubsystemAction("start mDNS server", startupSubsystemOptional, runtime.startMDNSServerFn),
		startupSubsystemOptionalStep("enable systemd-resolved", runtime.initializeResolvedFn),
		startupSubsystemRequiredStep("initialize broadcast subsystem", runtime.initializeBroadcastFn),
		startupSubsystemRequiredStep("initialize docker subsystem", runtime.initializeDockerFn),
		startupSubsystemOptionalStep("network reachability", func() error {
			if runtime.networkReachabilityFn == nil {
				return nil
			}
			internetAvailable := runtime.networkReachabilityFn(cloudCheckHost)
			zap.L().Info(fmt.Sprintf("Internet available: %t", internetAvailable))
			return nil
		}),
		startupSubsystemOptionalStep("swap configuration", func() error {
			return applySwapAndTmpDirSettings(runtime)
		}),
		startupSubsystemAction("prime rekor key", startupSubsystemOptional, runtime.primeRekorKeyFn),
	}
}

type startupRuntime struct {
	initializeConfigFn        func() error
	startConfigEventLoopFn    func(context.Context) error
	initializeAuthFn          func() error
	initializeRouterFn        func() error
	initializeSystemSupportFn func() error
	initializeExporterFn      func() error
	initializeImporterFn      func() error
	initializeBroadcastFn     func() error
	initializeResolvedFn      func() error
	initializeDockerFn        func() error
	startStartupContainersFn  func(bool)
	networkReachabilityFn     func(string) bool
	configureSwapFn           func(string, int) error
	setupTmpDirFn             func() error
	startMDNSServerFn         func() error
	initializeWiFiFn          func() error
	primeRekorKeyFn           func() error
}

func defaultStartupRuntime() startupRuntime {
	return startupRuntime{
		initializeConfigFn:     config.Initialize,
		startConfigEventLoopFn: func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel) },
		initializeAuthFn: func() error {
			auth.Initialize()
			return nil
		},
		initializeRouterFn: func() error {
			router.Initialize()
			return nil
		},
		initializeSystemSupportFn: func() error {
			groundSystem.InitializeSupport()
			return nil
		},
		initializeExporterFn:     func() error { return exporter.Initialize() },
		initializeImporterFn:     func() error { return importer.Initialize() },
		initializeBroadcastFn:    func() error { return startupdeps.InitializeBroadcast() },
		initializeResolvedFn:     func() error { return system.EnableResolved() },
		initializeDockerFn:       func() error { return startupdeps.NewStartupDockerRuntime().Initialize() },
		startStartupContainersFn: func(bool) {},
		networkReachabilityFn:    config.NetCheck,
		configureSwapFn:          system.ConfigureSwap,
		setupTmpDirFn:            func() error { return system.SetupTmpDir() },
		startMDNSServerFn:        func() error { routines.StartMDNSServer(); return nil },
		initializeWiFiFn:         func() error { return system.InitializeWiFi() },
		primeRekorKeyFn:          func() error { routines.PrimeRekorKey(); return nil },
	}
}

func (runtime startupRuntime) withDefaults(opts startupRuntime) startupRuntime {
	merged := runtime
	setIfNonNil(&merged.initializeConfigFn, opts.initializeConfigFn)
	setIfNonNil(&merged.startConfigEventLoopFn, opts.startConfigEventLoopFn)
	setIfNonNil(&merged.initializeAuthFn, opts.initializeAuthFn)
	setIfNonNil(&merged.initializeRouterFn, opts.initializeRouterFn)
	setIfNonNil(&merged.initializeSystemSupportFn, opts.initializeSystemSupportFn)
	setIfNonNil(&merged.initializeExporterFn, opts.initializeExporterFn)
	setIfNonNil(&merged.initializeImporterFn, opts.initializeImporterFn)
	setIfNonNil(&merged.initializeBroadcastFn, opts.initializeBroadcastFn)
	setIfNonNil(&merged.initializeResolvedFn, opts.initializeResolvedFn)
	setIfNonNil(&merged.initializeDockerFn, opts.initializeDockerFn)
	setIfNonNil(&merged.startStartupContainersFn, opts.startStartupContainersFn)
	setIfNonNil(&merged.networkReachabilityFn, opts.networkReachabilityFn)
	setIfNonNil(&merged.configureSwapFn, opts.configureSwapFn)
	setIfNonNil(&merged.setupTmpDirFn, opts.setupTmpDirFn)
	setIfNonNil(&merged.startMDNSServerFn, opts.startMDNSServerFn)
	setIfNonNil(&merged.initializeWiFiFn, opts.initializeWiFiFn)
	setIfNonNil(&merged.primeRekorKeyFn, opts.primeRekorKeyFn)
	return merged
}

func (runtime startBackgroundServicesRuntime) withDefaults(opts startBackgroundServicesRuntime) startBackgroundServicesRuntime {
	merged := runtime
	setIfNonNil(&merged.startVersionSubsystemFn, opts.startVersionSubsystemFn)
	setIfNonNil(&merged.startDockerSubsystemFn, opts.startDockerSubsystemFn)
	setIfNonNil(&merged.startUrbitTransitionHandlerFn, opts.startUrbitTransitionHandlerFn)
	setIfNonNil(&merged.startSystemTransitionHandlerFn, opts.startSystemTransitionHandlerFn)
	setIfNonNil(&merged.startNewShipTransitionHandlerFn, opts.startNewShipTransitionHandlerFn)
	setIfNonNil(&merged.startImportShipTransitionHandlerFn, opts.startImportShipTransitionHandlerFn)
	setIfNonNil(&merged.startRectifyUrbitFn, opts.startRectifyUrbitFn)
	setIfNonNil(&merged.syncRetrieveFn, opts.syncRetrieveFn)
	setIfNonNil(&merged.startLeakFn, opts.startLeakFn)
	setIfNonNil(&merged.startSysLogStreamerFn, opts.startSysLogStreamerFn)
	setIfNonNil(&merged.startDockerLogStreamerFn, opts.startDockerLogStreamerFn)
	setIfNonNil(&merged.startDockerLogConnRemoverFn, opts.startDockerLogConnRemoverFn)
	setIfNonNil(&merged.startOldLogsCleanerFn, opts.startOldLogsCleanerFn)
	setIfNonNil(&merged.startDiskUsageWarningFn, opts.startDiskUsageWarningFn)
	setIfNonNil(&merged.startSmartDiskCheckFn, opts.startSmartDiskCheckFn)
	setIfNonNil(&merged.startStartramRenewalReminderFn, opts.startStartramRenewalReminderFn)
	setIfNonNil(&merged.startPackScheduleLoopFn, opts.startPackScheduleLoopFn)
	setIfNonNil(&merged.startChopRoutinesFn, opts.startChopRoutinesFn)
	setIfNonNil(&merged.startBackupRoutinesFn, opts.startBackupRoutinesFn)
	return merged
}

func setIfNonNil[T any](target *T, value T) {
	if any(value) == nil {
		return
	}
	*target = value
}

type startBackgroundServicesRuntime struct {
	startVersionSubsystemFn            func(context.Context) error
	startDockerSubsystemFn             func(context.Context) error
	startUrbitTransitionHandlerFn      func(context.Context) error
	startSystemTransitionHandlerFn     func(context.Context) error
	startNewShipTransitionHandlerFn    func(context.Context) error
	startImportShipTransitionHandlerFn func(context.Context) error
	startRectifyUrbitFn                func(context.Context) error
	syncRetrieveFn                     func() error
	startLeakFn                        func(context.Context) error
	startSysLogStreamerFn              func(context.Context) error
	startDockerLogStreamerFn           func(context.Context) error
	startDockerLogConnRemoverFn        func(context.Context) error
	startOldLogsCleanerFn              func(context.Context) error
	startDiskUsageWarningFn            func(context.Context) error
	startSmartDiskCheckFn              func(context.Context) error
	startStartramRenewalReminderFn     func(context.Context) error
	startPackScheduleLoopFn            func(context.Context) error
	startChopRoutinesFn                func(context.Context) error
	startBackupRoutinesFn              func(context.Context) error
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

func (runtime startupRuntime) validate() error {
	missing := make([]string, 0, 10)
	if runtime.initializeConfigFn == nil {
		missing = append(missing, "initializeConfig")
	}
	if runtime.initializeAuthFn == nil {
		missing = append(missing, "initializeAuth")
	}
	if runtime.initializeRouterFn == nil {
		missing = append(missing, "initializeRouter")
	}
	if runtime.initializeSystemSupportFn == nil {
		missing = append(missing, "initializeSystemSupport")
	}
	if runtime.initializeExporterFn == nil {
		missing = append(missing, "initializeExporter")
	}
	if runtime.initializeImporterFn == nil {
		missing = append(missing, "initializeImporter")
	}
	if runtime.initializeBroadcastFn == nil {
		missing = append(missing, "initializeBroadcast")
	}
	if runtime.initializeDockerFn == nil {
		missing = append(missing, "initializeDocker")
	}
	if runtime.startStartupContainersFn == nil {
		missing = append(missing, "startStartupContainers")
	}
	if runtime.networkReachabilityFn == nil {
		missing = append(missing, "isNetworkReachable")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("startup runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	missing := make([]string, 0, 8)
	if runtime.startVersionSubsystemFn == nil {
		missing = append(missing, "startVersionSubsystem")
	}
	if runtime.startDockerSubsystemFn == nil {
		missing = append(missing, "startDockerSubsystem")
	}
	if runtime.startUrbitTransitionHandlerFn == nil {
		missing = append(missing, "startUrbitTransitionHandler")
	}
	if runtime.startSystemTransitionHandlerFn == nil {
		missing = append(missing, "startSystemTransitionHandler")
	}
	if runtime.startNewShipTransitionHandlerFn == nil {
		missing = append(missing, "startNewShipTransitionHandler")
	}
	if runtime.startImportShipTransitionHandlerFn == nil {
		missing = append(missing, "startImportShipTransitionHandler")
	}
	if runtime.startRectifyUrbitFn == nil {
		missing = append(missing, "startRectifyUrbit")
	}
	if runtime.startLeakFn == nil {
		missing = append(missing, "startLeak")
	}
	if runtime.startSysLogStreamerFn == nil {
		missing = append(missing, "startSysLogStreamer")
	}
	if runtime.startDockerLogStreamerFn == nil {
		missing = append(missing, "startDockerLogStreamer")
	}
	if runtime.startDockerLogConnRemoverFn == nil {
		missing = append(missing, "startDockerLogConnRemover")
	}
	if runtime.startOldLogsCleanerFn == nil {
		missing = append(missing, "startOldLogsCleaner")
	}
	if runtime.startDiskUsageWarningFn == nil {
		missing = append(missing, "startDiskUsageWarning")
	}
	if runtime.startSmartDiskCheckFn == nil {
		missing = append(missing, "startSmartDiskCheck")
	}
	if runtime.startPackScheduleLoopFn == nil {
		missing = append(missing, "startPackScheduleLoop")
	}
	if runtime.startChopRoutinesFn == nil {
		missing = append(missing, "startChopRoutines")
	}
	if runtime.startBackupRoutinesFn == nil {
		missing = append(missing, "startBackupRoutines")
	}
	if startramWgRegistered {
		if runtime.syncRetrieveFn == nil {
			missing = append(missing, "syncRetrieve")
		}
		if runtime.startStartramRenewalReminderFn == nil {
			missing = append(missing, "startStartramRenewalReminder")
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
	startC2cCheck  func(context.Context)
	startupRuntime startupRuntime
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
	StartC2cCheck  func(context.Context)
	StartupRuntime startupRuntime
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
		startupRuntime: options.StartupRuntime,
	}
	opts.startupRuntime = defaultStartupRuntime().withDefaults(opts.startupRuntime)
	if err := opts.startupRuntime.validate(); err != nil {
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
				if err := runStartupSubsystems(opts.startupRuntime.startupInitSubsystems()); err != nil {
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
				if opts.startupRuntime.startConfigEventLoopFn != nil {
					if err := opts.startupRuntime.startConfigEventLoopFn(ctx); err != nil {
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
				opts.startupRuntime.startStartupContainersFn(startramSettings.WgRegistered)
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

func applySwapAndTmpDirSettings(runtimeOps startupRuntime) error {
	swapSettings := config.SwapSettingsSnapshot()
	var startupErrs []error
	zap.L().Info(fmt.Sprintf("Setting up swap %v for %vG", swapSettings.SwapFile, swapSettings.SwapVal))
	if runtimeOps.configureSwapFn == nil {
		return nil
	}
	if err := runtimeOps.configureSwapFn(swapSettings.SwapFile, swapSettings.SwapVal); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
		startupErrs = append(startupErrs, fmt.Errorf("unable to set swap: %w", err))
	}
	zap.L().Info("Setting up /tmp directory")
	if runtimeOps.setupTmpDirFn == nil {
		if len(startupErrs) == 0 {
			return nil
		}
		return fmt.Errorf("unable to setup swap or /tmp: %w", errors.Join(startupErrs...))
	}
	if err := runtimeOps.setupTmpDirFn(); err != nil {
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
	case <-versionUpdate:
		zap.L().Info("Version info retrieved")
	case <-time.After(10 * time.Second):
		zap.L().Warn("Could not retrieve version info after 10 seconds!")
		versionStruct := config.LocalVersion()
		config.SetVersionChannel(versionStruct.Groundseg[updateBranch])
	}
}

func startBackgroundServices(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context)) (*startupBackgroundServices, error) {
	return startBackgroundServicesWithRuntime(ctx, startramWgRegistered, startC2cCheck, defaultStartBackgroundServicesRuntime())
}

func startBackgroundServicesWithRuntime(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context), runtime startBackgroundServicesRuntime) (*startupBackgroundServices, error) {
	runtime = defaultStartBackgroundServicesRuntime().withDefaults(runtime)
	if err := runtime.validate(startramWgRegistered); err != nil {
		return nil, err
	}
	services := &startupBackgroundServices{}
	if startC2cCheck != nil {
		services.add(superviseBackgroundService(ctx, "c2c-check", func(ctx context.Context) error {
			startC2cCheck(ctx)
			return nil
		}))
	}
	services.add(superviseBackgroundService(ctx, "version", runtime.startVersionSubsystemFn))
	services.add(superviseBackgroundService(ctx, "docker", runtime.startDockerSubsystemFn))
	services.add(superviseBackgroundService(ctx, "urbit-transition", runtime.startUrbitTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "system-transition", runtime.startSystemTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "new-ship-transition", runtime.startNewShipTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "import-ship-transition", runtime.startImportShipTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "rectify", runtime.startRectifyUrbitFn))
	if startramWgRegistered {
		syncHandle := superviseBackgroundService(ctx, "startram-sync", func(_ context.Context) error {
			return runtime.syncRetrieveFn()
		})
		services.add(syncHandle)
		select {
		case syncErr, ok := <-syncHandle.err:
			if ok && syncErr != nil {
				return nil, syncErr
			}
		default:
		}
	}
	services.add(superviseBackgroundService(ctx, "leak", runtime.startLeakFn))
	services.add(superviseBackgroundService(ctx, "sys-log-streamer", runtime.startSysLogStreamerFn))
	services.add(superviseBackgroundService(ctx, "docker-log-streamer", runtime.startDockerLogStreamerFn))
	services.add(superviseBackgroundService(ctx, "docker-log-conn-remover", runtime.startDockerLogConnRemoverFn))
	services.add(superviseBackgroundService(ctx, "old-logs-cleaner", runtime.startOldLogsCleanerFn))
	services.add(superviseBackgroundService(ctx, "disk-usage-warning", runtime.startDiskUsageWarningFn))
	services.add(superviseBackgroundService(ctx, "smart-disk-check", runtime.startSmartDiskCheckFn))
	if startramWgRegistered {
		services.add(superviseBackgroundService(ctx, "startram-renewal", runtime.startStartramRenewalReminderFn))
	}
	services.add(superviseBackgroundService(ctx, "pack-schedule", runtime.startPackScheduleLoopFn))
	services.add(superviseBackgroundService(ctx, "chop-routines", runtime.startChopRoutinesFn))
	services.add(superviseBackgroundService(ctx, "backup-routines", runtime.startBackupRoutinesFn))
	return services, nil
}

func defaultStartBackgroundServicesRuntime() startBackgroundServicesRuntime {
	return startBackgroundServicesRuntime{
		startVersionSubsystemFn:            routines.StartVersionSubsystemWithContext,
		startDockerSubsystemFn:             subsystem.StartDockerSubsystemWithContext,
		startUrbitTransitionHandlerFn:      rectify.UrbitTransitionHandlerWithContext,
		startSystemTransitionHandlerFn:     rectify.SystemTransitionHandlerWithContext,
		startNewShipTransitionHandlerFn:    rectify.NewShipTransitionHandlerWithContext,
		startImportShipTransitionHandlerFn: rectify.ImportShipTransitionHandlerWithContext,
		startRectifyUrbitFn:                rectify.RectifyUrbitWithContext,
		syncRetrieveFn:                     syncRetrieveWithStartram,
		startLeakFn:                        leak.StartLeakWithContext,
		startSysLogStreamerFn:              routines.SysLogStreamerWithContext,
		startDockerLogStreamerFn:           routines.DockerLogStreamerWithContext,
		startDockerLogConnRemoverFn:        routines.DockerLogConnRemoverWithContext,
		startOldLogsCleanerFn:              routines.OldLogsCleanerWithContext,
		startDiskUsageWarningFn:            routines.DiskUsageWarningWithContext,
		startSmartDiskCheckFn:              routines.SmartDiskCheckWithContext,
		startStartramRenewalReminderFn:     routines.StartramRenewalReminderWithContext,
		startPackScheduleLoopFn:            routines.PackScheduleLoopWithContext,
		startChopRoutinesFn:                routines.StartChopRoutinesWithContext,
		startBackupRoutinesFn:              routines.StartBackupRoutinesWithContext,
	}
}

func loadService(loadFn func() error, failureMessage string) {
	if loadFn == nil {
		zap.L().Warn("Startup load function is not configured")
		return
	}
	if err := loadFn(); err != nil {
		zap.L().Error(fmt.Sprintf("%s: %v", failureMessage, err))
	}
}
