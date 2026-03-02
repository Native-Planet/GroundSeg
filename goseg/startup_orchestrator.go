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
		startupSubsystemRequiredStep("initialize config subsystem", runtime.Initialization.InitializeConfigFn),
		startupSubsystemAction("initialize auth subsystem", startupSubsystemRequired, runtime.Initialization.InitializeAuthFn),
		startupSubsystemAction("initialize router subsystem", startupSubsystemRequired, runtime.Initialization.InitializeRouterFn),
		startupSubsystemAction("initialize system support subsystem", startupSubsystemRequired, runtime.Initialization.InitializeSystemSupportFn),
		startupSubsystemRequiredStep("initialize exporter subsystem", runtime.Initialization.InitializeExporterFn),
		startupSubsystemRequiredStep("initialize importer subsystem", runtime.Initialization.InitializeImporterFn),
		startupSubsystemOptionalStep("initialize wifi subsystem", runtime.Initialization.InitializeWiFiFn),
		startupSubsystemAction("start mDNS server", startupSubsystemOptional, runtime.Initialization.StartMDNSServerFn),
		startupSubsystemOptionalStep("enable systemd-resolved", runtime.Initialization.InitializeResolvedFn),
		startupSubsystemRequiredStep("initialize broadcast subsystem", runtime.Initialization.InitializeBroadcastFn),
		startupSubsystemRequiredStep("initialize docker subsystem", runtime.Initialization.InitializeDockerFn),
		startupSubsystemOptionalStep("network reachability", func() error {
			if runtime.Control.NetworkReachabilityFn == nil {
				return nil
			}
			internetAvailable := runtime.Control.NetworkReachabilityFn(cloudCheckHost)
			zap.L().Info(fmt.Sprintf("Internet available: %t", internetAvailable))
			return nil
		}),
		startupSubsystemOptionalStep("swap configuration", func() error {
			return applySwapAndTmpDirSettings(runtime)
		}),
		startupSubsystemAction("prime rekor key", startupSubsystemOptional, runtime.Initialization.PrimeRekorKeyFn),
	}
}

type startupRuntime struct {
	Initialization startupSubsystemInitializationRuntime
	Control        startupRuntimeControlRuntime
}

type startupSubsystemInitializationRuntime struct {
	InitializeConfigFn        func() error
	InitializeAuthFn          func() error
	InitializeRouterFn        func() error
	InitializeSystemSupportFn func() error
	InitializeExporterFn      func() error
	InitializeImporterFn      func() error
	InitializeBroadcastFn     func() error
	InitializeResolvedFn      func() error
	InitializeDockerFn        func() error
	StartMDNSServerFn         func() error
	InitializeWiFiFn          func() error
	PrimeRekorKeyFn           func() error
}

type startupRuntimeControlRuntime struct {
	StartConfigEventLoopFn   func(context.Context) error
	StartStartupContainersFn func(bool)
	NetworkReachabilityFn    func(string) bool
	ConfigureSwapFn          func(string, int) error
	SetupTmpDirFn            func() error
}

func defaultStartupRuntime() startupRuntime {
	return startupRuntime{
		Initialization: startupSubsystemInitializationRuntime{
			InitializeConfigFn: config.Initialize,
			InitializeAuthFn: func() error {
				auth.Initialize()
				return nil
			},
			InitializeRouterFn: func() error {
				router.Initialize()
				return nil
			},
			InitializeSystemSupportFn: func() error {
				groundSystem.InitializeSupport()
				return nil
			},
			InitializeExporterFn:  func() error { return exporter.Initialize() },
			InitializeImporterFn:  func() error { return importer.Initialize() },
			InitializeBroadcastFn: func() error { return startupdeps.InitializeBroadcast() },
			InitializeResolvedFn:  func() error { return system.EnableResolved() },
			InitializeDockerFn:    func() error { return startupdeps.NewStartupDockerRuntime().Initialize() },
			StartMDNSServerFn:     func() error { routines.StartMDNSServer(); return nil },
			InitializeWiFiFn:      func() error { return system.InitializeWiFi() },
			PrimeRekorKeyFn:       func() error { routines.PrimeRekorKey(); return nil },
		},
		Control: startupRuntimeControlRuntime{
			StartConfigEventLoopFn:   func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel) },
			StartStartupContainersFn: func(bool) {},
			NetworkReachabilityFn:    config.NetCheck,
			ConfigureSwapFn:          system.ConfigureSwap,
			SetupTmpDirFn:            func() error { return system.SetupTmpDir() },
		},
	}
}

func (runtime startupRuntime) withDefaults(opts startupRuntime) startupRuntime {
	return seams.Merge(runtime, opts)
}

func (runtime startBackgroundServicesRuntime) withDefaults(opts startBackgroundServicesRuntime) startBackgroundServicesRuntime {
	return seams.Merge(runtime, opts)
}

type startBackgroundServicesRuntime struct {
	Transition  transitionBackgroundServicesRuntime
	Streaming   streamingBackgroundServicesRuntime
	Maintenance maintenanceBackgroundServicesRuntime
	Startram    startramBackgroundServicesRuntime
}

type transitionBackgroundServicesRuntime struct {
	StartVersionSubsystemFn            func(context.Context) error
	StartDockerSubsystemFn             func(context.Context) error
	StartUrbitTransitionHandlerFn      func(context.Context) error
	StartSystemTransitionHandlerFn     func(context.Context) error
	StartNewShipTransitionHandlerFn    func(context.Context) error
	StartImportShipTransitionHandlerFn func(context.Context) error
	StartRectifyUrbitFn                func(context.Context) error
	StartLeakFn                        func(context.Context) error
}

type streamingBackgroundServicesRuntime struct {
	StartSysLogStreamerFn       func(context.Context) error
	StartDockerLogStreamerFn    func(context.Context) error
	StartDockerLogConnRemoverFn func(context.Context) error
}

type maintenanceBackgroundServicesRuntime struct {
	StartOldLogsCleanerFn   func(context.Context) error
	StartDiskUsageWarningFn func(context.Context) error
	StartSmartDiskCheckFn   func(context.Context) error
	StartPackScheduleLoopFn func(context.Context) error
	StartChopRoutinesFn     func(context.Context) error
	StartBackupRoutinesFn   func(context.Context) error
}

type startramBackgroundServicesRuntime struct {
	SyncRetrieveFn                 func() error
	StartStartramRenewalReminderFn func(context.Context) error
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

func (runtime startupRuntime) validate() error {
	var missing []string
	for _, err := range runtime.Initialization.validate() {
		missing = append(missing, err)
	}
	if err := runtime.Control.validate(); err != nil {
		missing = append(missing, err.Error())
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("startup runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startupSubsystemInitializationRuntime) validate() []string {
	missing := make([]string, 0, 8)
	if runtime.InitializeConfigFn == nil {
		missing = append(missing, "initializeConfig")
	}
	if runtime.InitializeAuthFn == nil {
		missing = append(missing, "initializeAuth")
	}
	if runtime.InitializeRouterFn == nil {
		missing = append(missing, "initializeRouter")
	}
	if runtime.InitializeSystemSupportFn == nil {
		missing = append(missing, "initializeSystemSupport")
	}
	if runtime.InitializeExporterFn == nil {
		missing = append(missing, "initializeExporter")
	}
	if runtime.InitializeImporterFn == nil {
		missing = append(missing, "initializeImporter")
	}
	if runtime.InitializeBroadcastFn == nil {
		missing = append(missing, "initializeBroadcast")
	}
	if runtime.InitializeDockerFn == nil {
		missing = append(missing, "initializeDocker")
	}
	return missing
}

func (runtime startupRuntimeControlRuntime) validate() error {
	missing := make([]string, 0, 2)
	if runtime.StartStartupContainersFn == nil {
		missing = append(missing, "startStartupContainers")
	}
	if runtime.NetworkReachabilityFn == nil {
		missing = append(missing, "isNetworkReachable")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("startup runtime control runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	var missing []string
	if err := runtime.Transition.validate(); err != nil {
		missing = append(missing, err.Error())
	}
	if err := runtime.Streaming.validate(); err != nil {
		missing = append(missing, err.Error())
	}
	if err := runtime.Maintenance.validate(); err != nil {
		missing = append(missing, err.Error())
	}
	if err := runtime.Startram.validate(startramWgRegistered); err != nil {
		missing = append(missing, err.Error())
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime transitionBackgroundServicesRuntime) validate() error {
	missing := make([]string, 0, 7)
	if runtime.StartVersionSubsystemFn == nil {
		missing = append(missing, "startVersionSubsystem")
	}
	if runtime.StartDockerSubsystemFn == nil {
		missing = append(missing, "startDockerSubsystem")
	}
	if runtime.StartUrbitTransitionHandlerFn == nil {
		missing = append(missing, "startUrbitTransitionHandler")
	}
	if runtime.StartSystemTransitionHandlerFn == nil {
		missing = append(missing, "startSystemTransitionHandler")
	}
	if runtime.StartNewShipTransitionHandlerFn == nil {
		missing = append(missing, "startNewShipTransitionHandler")
	}
	if runtime.StartImportShipTransitionHandlerFn == nil {
		missing = append(missing, "startImportShipTransitionHandler")
	}
	if runtime.StartRectifyUrbitFn == nil {
		missing = append(missing, "startRectifyUrbit")
	}
	if runtime.StartLeakFn == nil {
		missing = append(missing, "startLeak")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start transition background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime streamingBackgroundServicesRuntime) validate() error {
	missing := make([]string, 0, 3)
	if runtime.StartSysLogStreamerFn == nil {
		missing = append(missing, "startSysLogStreamer")
	}
	if runtime.StartDockerLogStreamerFn == nil {
		missing = append(missing, "startDockerLogStreamer")
	}
	if runtime.StartDockerLogConnRemoverFn == nil {
		missing = append(missing, "startDockerLogConnRemover")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start streaming background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime maintenanceBackgroundServicesRuntime) validate() error {
	missing := make([]string, 0, 6)
	if runtime.StartOldLogsCleanerFn == nil {
		missing = append(missing, "startOldLogsCleaner")
	}
	if runtime.StartDiskUsageWarningFn == nil {
		missing = append(missing, "startDiskUsageWarning")
	}
	if runtime.StartSmartDiskCheckFn == nil {
		missing = append(missing, "startSmartDiskCheck")
	}
	if runtime.StartPackScheduleLoopFn == nil {
		missing = append(missing, "startPackScheduleLoop")
	}
	if runtime.StartChopRoutinesFn == nil {
		missing = append(missing, "startChopRoutines")
	}
	if runtime.StartBackupRoutinesFn == nil {
		missing = append(missing, "startBackupRoutines")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start maintenance background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startramBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	if !startramWgRegistered {
		return nil
	}
	missing := make([]string, 0, 2)
	if runtime.SyncRetrieveFn == nil {
		missing = append(missing, "syncRetrieve")
	}
	if runtime.StartStartramRenewalReminderFn == nil {
		missing = append(missing, "startStartramRenewalReminder")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start startram background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
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
				if opts.startupRuntime.Control.StartConfigEventLoopFn != nil {
					if err := opts.startupRuntime.Control.StartConfigEventLoopFn(ctx); err != nil {
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
				opts.startupRuntime.Control.StartStartupContainersFn(startramSettings.WgRegistered)
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
	if runtimeOps.Control.ConfigureSwapFn == nil {
		return nil
	}
	if err := runtimeOps.Control.ConfigureSwapFn(swapSettings.SwapFile, swapSettings.SwapVal); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
		startupErrs = append(startupErrs, fmt.Errorf("unable to set swap: %w", err))
	}
	zap.L().Info("Setting up /tmp directory")
	if runtimeOps.Control.SetupTmpDirFn == nil {
		if len(startupErrs) == 0 {
			return nil
		}
		return fmt.Errorf("unable to setup swap or /tmp: %w", errors.Join(startupErrs...))
	}
	if err := runtimeOps.Control.SetupTmpDirFn(); err != nil {
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
	systemRuntime := session.LogstreamRuntimeState()
	logger.ConfigureLogstreamRuntime(systemRuntime)
	logstream.Configure(systemRuntime, systemRuntime.SystemLogMessages())
	services := &startupBackgroundServices{}
	if startC2cCheck != nil {
		services.add(superviseBackgroundService(ctx, "c2c-check", func(ctx context.Context) error {
			startC2cCheck(ctx)
			return nil
		}))
	}
	services.add(superviseBackgroundService(ctx, "version", runtime.Transition.StartVersionSubsystemFn))
	services.add(superviseBackgroundService(ctx, "docker", runtime.Transition.StartDockerSubsystemFn))
	services.add(superviseBackgroundService(ctx, "urbit-transition", runtime.Transition.StartUrbitTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "system-transition", runtime.Transition.StartSystemTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "new-ship-transition", runtime.Transition.StartNewShipTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "import-ship-transition", runtime.Transition.StartImportShipTransitionHandlerFn))
	services.add(superviseBackgroundService(ctx, "rectify", runtime.Transition.StartRectifyUrbitFn))
	if startramWgRegistered {
		syncHandle := superviseBackgroundService(ctx, "startram-sync", func(_ context.Context) error {
			return runtime.Startram.SyncRetrieveFn()
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
	services.add(superviseBackgroundService(ctx, "leak", runtime.Transition.StartLeakFn))
	services.add(superviseBackgroundService(ctx, "sys-log-streamer", runtime.Streaming.StartSysLogStreamerFn))
	services.add(superviseBackgroundService(ctx, "docker-log-streamer", runtime.Streaming.StartDockerLogStreamerFn))
	services.add(superviseBackgroundService(ctx, "docker-log-conn-remover", runtime.Streaming.StartDockerLogConnRemoverFn))
	services.add(superviseBackgroundService(ctx, "old-logs-cleaner", runtime.Maintenance.StartOldLogsCleanerFn))
	services.add(superviseBackgroundService(ctx, "disk-usage-warning", runtime.Maintenance.StartDiskUsageWarningFn))
	services.add(superviseBackgroundService(ctx, "smart-disk-check", runtime.Maintenance.StartSmartDiskCheckFn))
	if startramWgRegistered {
		services.add(superviseBackgroundService(ctx, "startram-renewal", runtime.Startram.StartStartramRenewalReminderFn))
	}
	services.add(superviseBackgroundService(ctx, "pack-schedule", runtime.Maintenance.StartPackScheduleLoopFn))
	services.add(superviseBackgroundService(ctx, "chop-routines", runtime.Maintenance.StartChopRoutinesFn))
	services.add(superviseBackgroundService(ctx, "backup-routines", runtime.Maintenance.StartBackupRoutinesFn))
	return services, nil
}

func defaultStartBackgroundServicesRuntime() startBackgroundServicesRuntime {
	return startBackgroundServicesRuntime{
		Transition: transitionBackgroundServicesRuntime{
			StartVersionSubsystemFn:            routines.StartVersionSubsystemWithContext,
			StartDockerSubsystemFn:             subsystem.StartDockerSubsystemWithContext,
			StartUrbitTransitionHandlerFn:      rectify.UrbitTransitionHandlerWithContext,
			StartSystemTransitionHandlerFn:     rectify.SystemTransitionHandlerWithContext,
			StartNewShipTransitionHandlerFn:    rectify.NewShipTransitionHandlerWithContext,
			StartImportShipTransitionHandlerFn: rectify.ImportShipTransitionHandlerWithContext,
			StartRectifyUrbitFn:                rectify.RectifyUrbitWithContext,
			StartLeakFn:                        leak.StartLeakWithContext,
		},
		Streaming: streamingBackgroundServicesRuntime{
			StartSysLogStreamerFn:       logstream.SysLogStreamerWithContext,
			StartDockerLogStreamerFn:    logstream.DockerLogStreamerWithContext,
			StartDockerLogConnRemoverFn: logstream.DockerLogConnRemoverWithContext,
		},
		Maintenance: maintenanceBackgroundServicesRuntime{
			StartOldLogsCleanerFn:   logstream.OldLogsCleanerWithContext,
			StartDiskUsageWarningFn: routines.DiskUsageWarningWithContext,
			StartSmartDiskCheckFn:   routines.SmartDiskCheckWithContext,
			StartPackScheduleLoopFn: routines.PackScheduleLoopWithContext,
			StartChopRoutinesFn:     routines.StartChopRoutinesWithContext,
			StartBackupRoutinesFn:   routines.StartBackupRoutinesWithContext,
		},
		Startram: startramBackgroundServicesRuntime{
			SyncRetrieveFn:                 syncRetrieveWithStartram,
			StartStartramRenewalReminderFn: routines.StartramRenewalReminderWithContext,
		},
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
