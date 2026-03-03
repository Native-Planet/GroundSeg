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
			logOptionalStartupFailure("init", step.name, err)
			return nil
		}
		return fmt.Errorf("%s initialization failed: %w", step.name, err)
	}
	return nil
}

func logOptionalStartupFailure(phase, subsystem string, err error) {
	if err == nil {
		return
	}
	zap.L().Warn(fmt.Sprintf("Optional startup %s subsystem failed: %s: %v", phase, subsystem, err))
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

type InitPhase interface {
	SubsystemSteps() []startupSubsystemStep
	MissingCallbacks() []string
}

type startupInitRuntime struct {
	startupInitCoreSubsystems    InitPhase
	startupInitHostSubsystems    InitPhase
	startupInitStorageSubsystems InitPhase
}

type startupInitCoreSubsystemRuntime struct {
	initializeConfigFn        startupInitCallbackFn
	initializeAuthFn          startupInitCallbackFn
	initializeRouterFn        startupInitCallbackFn
	initializeSystemSupportFn startupInitCallbackFn
	initializeExporterFn      startupInitCallbackFn
	initializeImporterFn      startupInitCallbackFn
	initializeBroadcastFn     startupInitCallbackFn
	initializeDockerFn        startupInitCallbackFn
}

func (runtime startupInitCoreSubsystemRuntime) SubsystemSteps() []startupSubsystemStep {
	return []startupSubsystemStep{
		{name: "initialize config subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeConfigFn},
		{name: "initialize auth subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeAuthFn},
		{name: "initialize router subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeRouterFn},
		{name: "initialize system support subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeSystemSupportFn},
		{name: "initialize exporter subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeExporterFn},
		{name: "initialize importer subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeImporterFn},
		{name: "initialize broadcast subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeBroadcastFn},
		{name: "initialize docker subsystem", policy: startupSubsystemRequired, initFn: runtime.initializeDockerFn},
	}
}

func (runtime startupInitCoreSubsystemRuntime) MissingCallbacks() []string {
	missing := []string{}
	if runtime.initializeConfigFn == nil {
		missing = append(missing, "initialize config subsystem")
	}
	if runtime.initializeAuthFn == nil {
		missing = append(missing, "initialize auth subsystem")
	}
	if runtime.initializeRouterFn == nil {
		missing = append(missing, "initialize router subsystem")
	}
	if runtime.initializeSystemSupportFn == nil {
		missing = append(missing, "initialize system support subsystem")
	}
	if runtime.initializeExporterFn == nil {
		missing = append(missing, "initialize exporter subsystem")
	}
	if runtime.initializeImporterFn == nil {
		missing = append(missing, "initialize importer subsystem")
	}
	if runtime.initializeBroadcastFn == nil {
		missing = append(missing, "initialize broadcast subsystem")
	}
	if runtime.initializeDockerFn == nil {
		missing = append(missing, "initialize docker subsystem")
	}
	return missing
}

type startupInitHostSubsystemRuntime struct {
	initializeWiFiFn      startupInitCallbackFn
	startMDNSServerFn     startupInitCallbackFn
	initializeResolvedFn  startupInitCallbackFn
	networkReachabilityFn func(string) bool
	primeRekorKeyFn       startupInitCallbackFn
}

func (runtime startupInitHostSubsystemRuntime) SubsystemSteps() []startupSubsystemStep {
	steps := []startupSubsystemStep{}
	steps = append(steps,
		startupSubsystemStep{name: "initialize wifi subsystem", policy: startupSubsystemOptional, initFn: runtime.initializeWiFiFn},
		startupSubsystemStep{name: "start mDNS server", policy: startupSubsystemOptional, initFn: runtime.startMDNSServerFn},
		startupSubsystemStep{name: "enable systemd-resolved", policy: startupSubsystemOptional, initFn: runtime.initializeResolvedFn},
	)
	if runtime.networkReachabilityFn != nil {
		steps = append(steps, startupSubsystemStep{name: "network reachability", policy: startupSubsystemOptional, initFn: networkReachabilityCallback(runtime.networkReachabilityFn)})
	}
	steps = append(steps,
		startupSubsystemStep{name: "prime rekor key", policy: startupSubsystemOptional, initFn: runtime.primeRekorKeyFn},
	)
	return steps
}

func (runtime startupInitHostSubsystemRuntime) MissingCallbacks() []string {
	return []string{}
}

type startupInitStorageSubsystemRuntime struct {
	configureSwapFn func(string, int) error
	setupTmpDirFn   func() error
}

func (runtime startupInitStorageSubsystemRuntime) SubsystemSteps() []startupSubsystemStep {
	if runtime.configureSwapFn == nil && runtime.setupTmpDirFn == nil {
		return nil
	}
	return []startupSubsystemStep{{
		name:   "swap configuration",
		policy: startupSubsystemOptional,
		initFn: func() error { return applySwapAndTmpDirSettings(&runtime) },
	}}
}

func (runtime startupInitStorageSubsystemRuntime) MissingCallbacks() []string {
	return []string{}
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
		startupInitCoreSubsystems: startupInitCoreSubsystemRuntime{
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
		},
		startupInitHostSubsystems: startupInitHostSubsystemRuntime{
			initializeWiFiFn: func() error { return system.InitializeWiFi() },
			startMDNSServerFn: func() error {
				routines.StartMDNSServer()
				return nil
			},
			initializeResolvedFn:  func() error { return system.EnableResolved() },
			networkReachabilityFn: config.NetCheck,
			primeRekorKeyFn:       func() error { routines.PrimeRekorKey(); return nil },
		},
		startupInitStorageSubsystems: startupInitStorageSubsystemRuntime{
			configureSwapFn: func(swapFile string, swapVal int) error {
				return system.ConfigureSwap(swapFile, swapVal)
			},
			setupTmpDirFn: func() error { return system.SetupTmpDir() },
		},
	}
}

func (runtime startupInitRuntime) withDefaults(opts startupInitRuntime) startupInitRuntime {
	if opts.startupInitCoreSubsystems == nil {
		opts.startupInitCoreSubsystems = runtime.startupInitCoreSubsystems
	}
	if opts.startupInitHostSubsystems == nil {
		opts.startupInitHostSubsystems = runtime.startupInitHostSubsystems
	}
	if opts.startupInitStorageSubsystems == nil {
		opts.startupInitStorageSubsystems = runtime.startupInitStorageSubsystems
	}
	return opts
}

func (runtime startupInitRuntime) initSubsystems() []startupSubsystemStep {
	subsystemSteps := []startupSubsystemStep{}
	if runtime.startupInitCoreSubsystems != nil {
		subsystemSteps = append(subsystemSteps, runtime.startupInitCoreSubsystems.SubsystemSteps()...)
	}
	if runtime.startupInitHostSubsystems != nil {
		subsystemSteps = append(subsystemSteps, runtime.startupInitHostSubsystems.SubsystemSteps()...)
	}
	if runtime.startupInitStorageSubsystems != nil {
		subsystemSteps = append(subsystemSteps, runtime.startupInitStorageSubsystems.SubsystemSteps()...)
	}
	return subsystemSteps
}

func (runtime startupInitRuntime) validate() []string {
	missing := []string{}
	if runtime.startupInitCoreSubsystems == nil {
		missing = append(missing, "initialize core subsystem callbacks")
	} else {
		missing = append(missing, runtime.startupInitCoreSubsystems.MissingCallbacks()...)
	}
	if runtime.startupInitHostSubsystems == nil {
		missing = append(missing, "initialize host subsystem callbacks")
	}
	if runtime.startupInitStorageSubsystems == nil {
		missing = append(missing, "initialize storage subsystem callbacks")
	}
	return missing
}

type ServerBootstrap interface {
	StartConfigEventLoop(context.Context) error
	StartStartupContainers(startramWgRegistered bool)
	Validate() []string
}

type startupBootstrapRuntime struct {
	bootstrap ServerBootstrap
}

type startupBootstrapRuntimeFns struct {
	startConfigEventLoopFn   func(context.Context) error
	startStartupContainersFn func(bool)
}

func (runtime startupBootstrapRuntimeFns) StartConfigEventLoop(ctx context.Context) error {
	if runtime.startConfigEventLoopFn == nil {
		return nil
	}
	return runtime.startConfigEventLoopFn(ctx)
}

func (runtime startupBootstrapRuntimeFns) StartStartupContainers(startramWgRegistered bool) {
	if runtime.startStartupContainersFn != nil {
		runtime.startStartupContainersFn(startramWgRegistered)
	}
}

func (runtime startupBootstrapRuntimeFns) Validate() []string {
	if runtime.startStartupContainersFn == nil {
		return []string{"startStartupContainers"}
	}
	return nil
}

func (runtime startupBootstrapRuntime) StartConfigEventLoop(ctx context.Context) error {
	if runtime.bootstrap == nil {
		return nil
	}
	return runtime.bootstrap.StartConfigEventLoop(ctx)
}

func (runtime startupBootstrapRuntime) StartStartupContainers(startramWgRegistered bool) {
	if runtime.bootstrap == nil {
		return
	}
	runtime.bootstrap.StartStartupContainers(startramWgRegistered)
}

func (runtime startupBootstrapRuntime) withDefaults(opts startupBootstrapRuntime) startupBootstrapRuntime {
	if opts.bootstrap == nil {
		opts.bootstrap = runtime.bootstrap
	}
	return opts
}

func (runtime startupBootstrapRuntime) validate() []string {
	if runtime.bootstrap == nil {
		return []string{"startStartupContainers"}
	}
	return runtime.bootstrap.Validate()
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
			bootstrap: startupBootstrapRuntimeFns{
				startConfigEventLoopFn:   func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel()) },
				startStartupContainersFn: func(bool) {},
			},
		},
	}
}

func (runtime StartupRuntime) withDefaults(opts StartupRuntime) StartupRuntime {
	runtime.startupInitRuntime = runtime.startupInitRuntime.withDefaults(opts.startupInitRuntime)
	runtime.startupBootstrapRuntime = runtime.startupBootstrapRuntime.withDefaults(opts.startupBootstrapRuntime)
	return runtime
}

type BackgroundServiceGroup interface {
	ServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec
	MissingCallbacks(startramWgRegistered bool) []string
}

type startBackgroundServicesRuntime struct {
	startupBackgroundCoreSubsystems      BackgroundServiceGroup
	startupBackgroundShipSubsystems      BackgroundServiceGroup
	startupBackgroundStartramSubsystems  BackgroundServiceGroup
	startupBackgroundLogstreamSubsystems BackgroundServiceGroup
}

type backgroundSubsystemRuntime struct {
	startVersionSubsystemFn func(context.Context) error
	startDockerSubsystemFn  func(context.Context) error
	startLeakFn             func(context.Context) error
	startSysLogStreamerFn   func(context.Context) error
	startOldLogsCleanerFn   func(context.Context) error
	startDiskUsageWarningFn func(context.Context) error
	startSmartDiskCheckFn   func(context.Context) error
	startPackScheduleLoopFn func(context.Context) error
	startChopRoutinesFn     func(context.Context) error
	startBackupRoutinesFn   func(context.Context) error
}

func (runtime backgroundSubsystemRuntime) ServiceSpecs() []backgroundServiceSpec {
	return []backgroundServiceSpec{
		{name: "version", startFn: runtime.startVersionSubsystemFn, failFast: false},
		{name: "docker", startFn: runtime.startDockerSubsystemFn, failFast: false},
		{name: "leak", startFn: runtime.startLeakFn, failFast: false},
		{name: "sys-log-streamer", startFn: runtime.startSysLogStreamerFn, failFast: false},
		{name: "old-logs-cleaner", startFn: runtime.startOldLogsCleanerFn, failFast: false},
		{name: "disk-usage-warning", startFn: runtime.startDiskUsageWarningFn, failFast: false},
		{name: "smart-disk-check", startFn: runtime.startSmartDiskCheckFn, failFast: false},
		{name: "pack-schedule", startFn: runtime.startPackScheduleLoopFn, failFast: false},
		{name: "chop-routines", startFn: runtime.startChopRoutinesFn, failFast: false},
		{name: "backup-routines", startFn: runtime.startBackupRoutinesFn, failFast: false},
	}
}

func (runtime backgroundSubsystemRuntime) MissingCallbacks(_ bool) []string {
	missing := []string{}
	if runtime.startVersionSubsystemFn == nil {
		missing = append(missing, "version")
	}
	if runtime.startDockerSubsystemFn == nil {
		missing = append(missing, "docker")
	}
	if runtime.startLeakFn == nil {
		missing = append(missing, "leak")
	}
	if runtime.startSysLogStreamerFn == nil {
		missing = append(missing, "sys-log-streamer")
	}
	if runtime.startOldLogsCleanerFn == nil {
		missing = append(missing, "old-logs-cleaner")
	}
	if runtime.startDiskUsageWarningFn == nil {
		missing = append(missing, "disk-usage-warning")
	}
	if runtime.startSmartDiskCheckFn == nil {
		missing = append(missing, "smart-disk-check")
	}
	if runtime.startPackScheduleLoopFn == nil {
		missing = append(missing, "pack-schedule")
	}
	if runtime.startChopRoutinesFn == nil {
		missing = append(missing, "chop-routines")
	}
	if runtime.startBackupRoutinesFn == nil {
		missing = append(missing, "backup-routines")
	}
	return missing
}

type startupBackgroundCoreSubsystemRuntime backgroundSubsystemRuntime

func (runtime startupBackgroundCoreSubsystemRuntime) ServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	return backgroundSubsystemRuntime(runtime).ServiceSpecs()
}

func (runtime startupBackgroundCoreSubsystemRuntime) MissingCallbacks(startramWgRegistered bool) []string {
	return backgroundSubsystemRuntime(runtime).MissingCallbacks(startramWgRegistered)
}

func (runtime StartupRuntime) validate() error {
	missing := runtime.startupInitRuntime.validate()
	missing = append(missing, runtime.startupBootstrapRuntime.validate()...)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("startup runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

type startupBackgroundShipSubsystemRuntime struct {
	startUrbitTransitionFn      func(context.Context) error
	startSystemTransitionFn     func(context.Context) error
	startNewShipTransitionFn    func(context.Context) error
	startImportShipTransitionFn func(context.Context) error
	startRectifyUrbitFn         func(context.Context) error
}

func (runtime startupBackgroundShipSubsystemRuntime) ServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	return []backgroundServiceSpec{
		{name: "urbit-transition", startFn: runtime.startUrbitTransitionFn, failFast: false},
		{name: "system-transition", startFn: runtime.startSystemTransitionFn, failFast: false},
		{name: "new-ship-transition", startFn: runtime.startNewShipTransitionFn, failFast: false},
		{name: "import-ship-transition", startFn: runtime.startImportShipTransitionFn, failFast: false},
		{name: "rectify", startFn: runtime.startRectifyUrbitFn, failFast: false},
	}
}

func (runtime startupBackgroundShipSubsystemRuntime) MissingCallbacks(_ bool) []string {
	missing := []string{}
	if runtime.startUrbitTransitionFn == nil {
		missing = append(missing, "urbit-transition")
	}
	if runtime.startSystemTransitionFn == nil {
		missing = append(missing, "system-transition")
	}
	if runtime.startNewShipTransitionFn == nil {
		missing = append(missing, "new-ship-transition")
	}
	if runtime.startImportShipTransitionFn == nil {
		missing = append(missing, "import-ship-transition")
	}
	if runtime.startRectifyUrbitFn == nil {
		missing = append(missing, "rectify")
	}
	return missing
}

type startupBackgroundStartramSubsystemRuntime struct {
	syncRetrieveFn                 func(context.Context) error
	startStartramRenewalReminderFn func(context.Context) error
}

func (runtime startupBackgroundStartramSubsystemRuntime) ServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	if !startramWgRegistered {
		return nil
	}
	return []backgroundServiceSpec{
		{name: "startram-sync", startFn: runtime.syncRetrieveFn, failFast: true},
		{name: "startram-renewal", startFn: runtime.startStartramRenewalReminderFn, failFast: false},
	}
}

func (runtime startupBackgroundStartramSubsystemRuntime) MissingCallbacks(startramWgRegistered bool) []string {
	if !startramWgRegistered {
		return []string{}
	}
	missing := []string{}
	if runtime.syncRetrieveFn == nil {
		missing = append(missing, "startram-sync")
	}
	if runtime.startStartramRenewalReminderFn == nil {
		missing = append(missing, "startram-renewal")
	}
	return missing
}

type startupBackgroundLogstreamSubsystemRuntime struct {
	startDockerLogStreamerFn    func(context.Context) error
	startDockerLogConnRemoverFn func(context.Context) error
}

func (runtime startupBackgroundLogstreamSubsystemRuntime) ServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	return []backgroundServiceSpec{
		{name: "docker-log-streamer", startFn: runtime.startDockerLogStreamerFn, failFast: false},
		{name: "docker-log-conn-remover", startFn: runtime.startDockerLogConnRemoverFn, failFast: false},
	}
}

func (runtime startupBackgroundLogstreamSubsystemRuntime) MissingCallbacks(_ bool) []string {
	missing := []string{}
	if runtime.startDockerLogStreamerFn == nil {
		missing = append(missing, "docker-log-streamer")
	}
	if runtime.startDockerLogConnRemoverFn == nil {
		missing = append(missing, "docker-log-conn-remover")
	}
	return missing
}

type backgroundServiceSpec struct {
	name     string
	startFn  func(context.Context) error
	failFast bool
}

func (runtime startBackgroundServicesRuntime) withDefaults(opts startBackgroundServicesRuntime) startBackgroundServicesRuntime {
	if opts.startupBackgroundCoreSubsystems == nil {
		opts.startupBackgroundCoreSubsystems = runtime.startupBackgroundCoreSubsystems
	}
	if opts.startupBackgroundShipSubsystems == nil {
		opts.startupBackgroundShipSubsystems = runtime.startupBackgroundShipSubsystems
	}
	if opts.startupBackgroundStartramSubsystems == nil {
		opts.startupBackgroundStartramSubsystems = runtime.startupBackgroundStartramSubsystems
	}
	if opts.startupBackgroundLogstreamSubsystems == nil {
		opts.startupBackgroundLogstreamSubsystems = runtime.startupBackgroundLogstreamSubsystems
	}
	return opts
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

func (runtime startBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	missing := runtime.backgroundServiceMissingCallbacks(startramWgRegistered)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("start background services runtime missing required callbacks: %s", strings.Join(missing, ", "))
}

func (runtime startBackgroundServicesRuntime) backgroundServiceMissingCallbacks(startramWgRegistered bool) []string {
	missing := []string{}
	if runtime.startupBackgroundCoreSubsystems == nil {
		missing = append(missing, "core startup services")
	} else {
		missing = append(missing, runtime.startupBackgroundCoreSubsystems.MissingCallbacks(startramWgRegistered)...)
	}
	if runtime.startupBackgroundShipSubsystems == nil {
		missing = append(missing, "ship transition services")
	} else {
		missing = append(missing, runtime.startupBackgroundShipSubsystems.MissingCallbacks(startramWgRegistered)...)
	}
	if runtime.startupBackgroundLogstreamSubsystems == nil {
		missing = append(missing, "docker logstream services")
	} else {
		missing = append(missing, runtime.startupBackgroundLogstreamSubsystems.MissingCallbacks(startramWgRegistered)...)
	}
	if runtime.startupBackgroundStartramSubsystems == nil {
		missing = append(missing, "startram services")
	} else {
		missing = append(missing, runtime.startupBackgroundStartramSubsystems.MissingCallbacks(startramWgRegistered)...)
	}
	return missing
}

func (runtime startBackgroundServicesRuntime) startupServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	specs := []backgroundServiceSpec{}
	if runtime.startupBackgroundCoreSubsystems != nil {
		specs = append(specs, runtime.startupBackgroundCoreSubsystems.ServiceSpecs(startramWgRegistered)...)
	}
	if runtime.startupBackgroundShipSubsystems != nil {
		specs = append(specs, runtime.startupBackgroundShipSubsystems.ServiceSpecs(startramWgRegistered)...)
	}
	if runtime.startupBackgroundStartramSubsystems != nil {
		specs = append(specs, runtime.startupBackgroundStartramSubsystems.ServiceSpecs(startramWgRegistered)...)
	}
	if runtime.startupBackgroundLogstreamSubsystems != nil {
		specs = append(specs, runtime.startupBackgroundLogstreamSubsystems.ServiceSpecs(startramWgRegistered)...)
	}
	return specs
}

func resolveStartupRuntime(runtime StartupRuntime) (StartupRuntime, error) {
	resolvedRuntime := defaultStartupRuntime().withDefaults(runtime)
	if err := resolvedRuntime.validate(); err != nil {
		return resolvedRuntime, err
	}
	return resolvedRuntime, nil
}

func resolveStartBackgroundServicesRuntime(
	logstreamRuntime *logstream.LogstreamRuntime,
	startramWgRegistered bool,
	runtime startBackgroundServicesRuntime,
) (startBackgroundServicesRuntime, error) {
	resolved := defaultStartBackgroundServicesRuntime(logstreamRuntime).withDefaults(runtime)
	if err := resolved.validate(startramWgRegistered); err != nil {
		return resolved, err
	}
	return resolved, nil
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

func (services *startupBackgroundServices) add(service backgroundServiceHandle) {
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
			logOptionalStartupFailure("start", task.name, err)
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
			logOptionalStartupFailure("health", task.name, err)
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
	resolvedRuntime, err := resolveStartupRuntime(opts.StartupRuntime)
	if err != nil {
		return err
	}
	opts.StartupRuntime = resolvedRuntime
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
				if err := opts.StartupRuntime.StartConfigEventLoop(ctx); err != nil {
					return fmt.Errorf("start config event loop failed: %w", err)
				}
				startramSettings := config.StartramSettingsSnapshot()
				versionUpdateChannel, remoteVersion := startVersionDiscovery(updateSettings.UpdateMode, updateSettings.UpdateBranch)
				services, err := startBackgroundServices(ctx, startramSettings.WgRegistered, opts.startC2cCheck)
				if err != nil {
					return err
				}
				backgroundServices = services

				waitForVersionDiscovery(remoteVersion, versionUpdateChannel, updateSettings.UpdateBranch)
				opts.StartupRuntime.StartStartupContainers(startramSettings.WgRegistered)
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

func applySwapAndTmpDirSettings(storageOps *startupInitStorageSubsystemRuntime) error {
	swapSettings := config.SwapSettingsSnapshot()
	var startupErrs []error
	zap.L().Info(fmt.Sprintf("Setting up swap %v for %vG", swapSettings.SwapFile, swapSettings.SwapVal))
	if storageOps.configureSwapFn != nil {
		if err := storageOps.configureSwapFn(swapSettings.SwapFile, swapSettings.SwapVal); err != nil {
			zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
			startupErrs = append(startupErrs, fmt.Errorf("unable to set swap: %w", err))
		}
	}
	zap.L().Info("Setting up /tmp directory")
	if storageOps.setupTmpDirFn != nil {
		if err := storageOps.setupTmpDirFn(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to setup /tmp: %v", err))
			startupErrs = append(startupErrs, fmt.Errorf("unable to setup /tmp: %w", err))
		}
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
	return startBackgroundServicesWithRuntime(ctx, startramWgRegistered, startC2cCheck, defaultStartBackgroundServicesRuntime(nil))
}

func startBackgroundServicesWithRuntime(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context) error, runtime startBackgroundServicesRuntime) (*startupBackgroundServices, error) {
	systemRuntime := session.LogstreamRuntimeState()
	logger.ConfigureLogstreamRuntime(systemRuntime)
	logstreamRuntime := logstream.Configure(systemRuntime, systemRuntime.SystemLogMessages())
	runtime, err := resolveStartBackgroundServicesRuntime(logstreamRuntime, startramWgRegistered, runtime)
	if err != nil {
		return nil, err
	}
	services := &startupBackgroundServices{}
	if startC2cCheck != nil {
		services.add(superviseBackgroundService(ctx, "c2c-check", func(ctx context.Context) error {
			return startC2cCheck(ctx)
		}))
	}
	startService := func(name string, startFn func(context.Context) error, failFast bool) error {
		handle := superviseBackgroundService(ctx, name, startFn)
		services.add(handle)
		if !failFast {
			return nil
		}
		select {
		case serviceErr, ok := <-handle.err:
			if ok && serviceErr != nil {
				return serviceErr
			}
		default:
		}
		return nil
	}

	for _, service := range runtime.startupServiceSpecs(startramWgRegistered) {
		if err := startService(service.name, service.startFn, service.failFast); err != nil {
			return nil, err
		}
	}
	return services, nil
}

func defaultStartBackgroundServicesRuntime(logstreamRuntime *logstream.LogstreamRuntime) startBackgroundServicesRuntime {
	return startBackgroundServicesRuntime{
		startupBackgroundCoreSubsystems: &startupBackgroundCoreSubsystemRuntime{
			startVersionSubsystemFn: routines.StartVersionSubsystemWithContext,
			startDockerSubsystemFn:  subsystem.StartDockerSubsystem,
			startLeakFn:             leak.StartLeakWithContext,
			startSysLogStreamerFn:   func(ctx context.Context) error { return logstream.SysLogStreamerWithRuntime(ctx, logstreamRuntime) },
			startOldLogsCleanerFn:   logstream.OldLogsCleanerWithContext,
			startDiskUsageWarningFn: routines.DiskUsageWarningWithContext,
			startSmartDiskCheckFn:   routines.SmartDiskCheckWithContext,
			startPackScheduleLoopFn: routines.PackScheduleLoopWithContext,
			startChopRoutinesFn:     routines.StartChopRoutinesWithContext,
			startBackupRoutinesFn:   routines.StartBackupRoutinesWithContext,
		},
		startupBackgroundShipSubsystems: &startupBackgroundShipSubsystemRuntime{
			startUrbitTransitionFn:      rectify.UrbitTransitionHandlerWithContext,
			startSystemTransitionFn:     rectify.SystemTransitionHandlerWithContext,
			startNewShipTransitionFn:    rectify.NewShipTransitionHandlerWithContext,
			startImportShipTransitionFn: rectify.ImportShipTransitionHandlerWithContext,
			startRectifyUrbitFn:         rectify.RectifyUrbitWithContext,
		},
		startupBackgroundStartramSubsystems: &startupBackgroundStartramSubsystemRuntime{
			syncRetrieveFn: func(context.Context) error {
				return syncRetrieveWithStartram()
			},
			startStartramRenewalReminderFn: routines.StartramRenewalReminderWithContext,
		},
		startupBackgroundLogstreamSubsystems: &startupBackgroundLogstreamSubsystemRuntime{
			startDockerLogStreamerFn: func(ctx context.Context) error { return logstream.DockerLogStreamerWithRuntime(ctx, logstreamRuntime) },
			startDockerLogConnRemoverFn: func(ctx context.Context) error {
				return logstream.DockerLogConnRemoverWithRuntime(ctx, logstreamRuntime)
			},
		},
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
