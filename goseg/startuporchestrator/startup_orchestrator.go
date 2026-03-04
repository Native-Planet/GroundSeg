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

type startupInitCoreRuntime struct {
	InitializeConfigFn        func() error `runtime:"startup-init" runtime_name:"initializeConfig"`
	InitializeAuthFn          func() error `runtime:"startup-init" runtime_name:"initializeAuth"`
	InitializeRouterFn        func() error `runtime:"startup-init" runtime_name:"initializeRouter"`
	InitializeSystemSupportFn func() error `runtime:"startup-init" runtime_name:"initializeSystemSupport"`
	InitializeExporterFn      func() error `runtime:"startup-init" runtime_name:"initializeExporter"`
	InitializeImporterFn      func() error `runtime:"startup-init" runtime_name:"initializeImporter"`
	InitializeBroadcastFn     func() error `runtime:"startup-init" runtime_name:"initializeBroadcast"`
	InitializeDockerFn        func() error `runtime:"startup-init" runtime_name:"initializeDocker"`
}

type startupInitHostRuntime struct {
	InitializeWiFiFn      func() error      `runtime:"startup-init" runtime_name:"initializeWiFi"`
	StartMDNSServerFn     func() error      `runtime:"startup-init" runtime_name:"startMDNSServer"`
	InitializeResolvedFn  func() error      `runtime:"startup-init" runtime_name:"initializeResolved"`
	NetworkReachabilityFn func(string) bool `runtime:"startup-init" runtime_name:"networkReachability"`
	PrimeRekorKeyFn       func() error      `runtime:"startup-init" runtime_name:"primeRekorKey"`
}

type startupInitStorageRuntime struct {
	ConfigureSwapFn func(string, int) error    `runtime:"startup-init" runtime_name:"configureSwap"`
	SwapSettingsFn  func() config.SwapSettings `runtime:"startup-init"`
	SetupTmpDirFn   func() error               `runtime:"startup-init" runtime_name:"setupTmpDir"`
}

type startupInitRuntime struct {
	startupInitCoreRuntime
	startupInitHostRuntime
	startupInitStorageRuntime
}

func (runtime startupInitRuntime) initSubsystems() []startupSubsystemStep {
	return []startupSubsystemStep{
		{name: "init config", policy: startupSubsystemRequired, initFn: runtime.InitializeConfigFn},
		{name: "init auth", policy: startupSubsystemRequired, initFn: runtime.InitializeAuthFn},
		{name: "init router", policy: startupSubsystemRequired, initFn: runtime.InitializeRouterFn},
		{name: "init system support", policy: startupSubsystemRequired, initFn: runtime.InitializeSystemSupportFn},
		{name: "init exporter", policy: startupSubsystemRequired, initFn: runtime.InitializeExporterFn},
		{name: "init importer", policy: startupSubsystemRequired, initFn: runtime.InitializeImporterFn},
		{name: "init broadcast", policy: startupSubsystemRequired, initFn: runtime.InitializeBroadcastFn},
		{name: "init docker", policy: startupSubsystemRequired, initFn: runtime.InitializeDockerFn},
		{name: "init wifi", policy: startupSubsystemOptional, initFn: runtime.InitializeWiFiFn},
		{
			name:   "start mdns server",
			policy: startupSubsystemOptional,
			initFn: runtime.StartMDNSServerFn,
		},
		{name: "init resolved", policy: startupSubsystemOptional, initFn: runtime.InitializeResolvedFn},
		{
			name:   "network reachability",
			policy: startupSubsystemOptional,
			initFn: func() error {
				if runtime.NetworkReachabilityFn == nil {
					return nil
				}
				if runtime.NetworkReachabilityFn(cloudCheckHost) {
					return nil
				}
				return fmt.Errorf("network reachability check failed")
			},
		},
		{name: "prime rekor key", policy: startupSubsystemOptional, initFn: runtime.PrimeRekorKeyFn},
		{
			name:   "configure swap",
			policy: startupSubsystemRequired,
			initFn: func() error {
				conf := config.SwapSettings{}
				if runtime.SwapSettingsFn != nil {
					conf = runtime.SwapSettingsFn()
				}
				if conf.SwapFile == "" || conf.SwapVal == 0 {
					return nil
				}
				if runtime.ConfigureSwapFn == nil {
					return nil
				}
				return runtime.ConfigureSwapFn(conf.SwapFile, conf.SwapVal)
			},
		},
		{name: "setup tmp dir", policy: startupSubsystemRequired, initFn: runtime.SetupTmpDirFn},
	}
}

func (runtime startupInitRuntime) validate() error {
	return seams.NewCallbackRequirementsWithGroups("startup-init").ValidateCallbacks(runtime, "startup init runtime")
}

func startupInitDefaultRuntime() startupInitRuntime {
	return startupInitRuntime{
		startupInitCoreRuntime: startupInitCoreRuntime{
			InitializeConfigFn: func() error { return config.Initialize() },
			InitializeAuthFn:   func() error { auth.Initialize(); return nil },
			InitializeRouterFn: func() error { router.Initialize(); return nil },
			InitializeSystemSupportFn: func() error {
				groundSystem.InitializeSupport()
				return nil
			},
			InitializeExporterFn:  func() error { return exporter.Initialize() },
			InitializeImporterFn:  func() error { return importer.Initialize() },
			InitializeBroadcastFn: func() error { return startupdeps.InitializeBroadcast() },
			InitializeDockerFn:    func() error { return startupdeps.NewStartupDockerRuntime().Initialize() },
		},
		startupInitHostRuntime: startupInitHostRuntime{
			InitializeWiFiFn:      func() error { return system.InitializeWiFi() },
			StartMDNSServerFn:     func() error { routines.StartMDNSServer(); return nil },
			InitializeResolvedFn:  func() error { return system.EnableResolved() },
			NetworkReachabilityFn: config.NetCheck,
			PrimeRekorKeyFn:       func() error { routines.PrimeRekorKey(); return nil },
		},
		startupInitStorageRuntime: startupInitStorageRuntime{
			ConfigureSwapFn: func(swapFile string, swapVal int) error { return system.ConfigureSwap(swapFile, swapVal) },
			SwapSettingsFn:  config.SwapSettingsSnapshot,
			SetupTmpDirFn:   func() error { return system.SetupTmpDir() },
		},
	}
}

type StartupRuntime struct {
	startupInitRuntime
	StartConfigEventLoopFn    func(context.Context) error    `runtime:"startup-runtime" runtime_name:"startConfigEventLoop"`
	StartStartupContainersFn  func(bool)                     `runtime:"startup-runtime" runtime_name:"startStartupContainers"`
	StartupUpdateSettingsFn   func() config.UpdateSettings   `runtime:"startup-runtime" runtime_name:"startupUpdateSettings"`
	StartupStartramSettingsFn func() config.StartramSettings `runtime:"startup-runtime" runtime_name:"startupStartramSettings"`
}

func defaultStartupRuntime() StartupRuntime {
	return StartupRuntime{
		startupInitRuntime:        startupInitDefaultRuntime(),
		StartConfigEventLoopFn:    func(ctx context.Context) error { return config.StartConfEventLoop(ctx, system.ConfChannel()) },
		StartStartupContainersFn:  func(bool) {},
		StartupUpdateSettingsFn:   config.UpdateSettingsSnapshot,
		StartupStartramSettingsFn: config.StartramSettingsSnapshot,
	}
}

func (runtime StartupRuntime) startupInitSubsystems() []startupSubsystemStep {
	return runtime.startupInitRuntime.initSubsystems()
}

type startBackgroundServicesRuntime struct {
	StartVersionFn              func(context.Context) error `runtime:"startup-background-service" runtime_name:"version"`
	StartDockerFn               func(context.Context) error `runtime:"startup-background-service" runtime_name:"docker"`
	StartLeakFn                 func(context.Context) error `runtime:"startup-background-service" runtime_name:"leak"`
	StartSysLogStreamerFn       func(context.Context) error `runtime:"startup-background-service" runtime_name:"sys-log-streamer"`
	StartOldLogsCleanerFn       func(context.Context) error `runtime:"startup-background-service" runtime_name:"old-logs-cleaner"`
	StartDiskUsageWarningFn     func(context.Context) error `runtime:"startup-background-service" runtime_name:"disk-usage-warning"`
	StartSmartDiskCheckFn       func(context.Context) error `runtime:"startup-background-service" runtime_name:"smart-disk-check"`
	StartPackScheduleFn         func(context.Context) error `runtime:"startup-background-service" runtime_name:"pack-schedule"`
	StartChopRoutinesFn         func(context.Context) error `runtime:"startup-background-service" runtime_name:"chop-routines"`
	StartBackupRoutinesFn       func(context.Context) error `runtime:"startup-background-service" runtime_name:"backup-routines"`
	StartUrbitTransitionFn      func(context.Context) error `runtime:"startup-background-service" runtime_name:"urbit-transition"`
	StartSystemTransitionFn     func(context.Context) error `runtime:"startup-background-service" runtime_name:"system-transition"`
	StartNewShipTransitionFn    func(context.Context) error `runtime:"startup-background-service" runtime_name:"new-ship-transition"`
	StartImportShipTransitionFn func(context.Context) error `runtime:"startup-background-service" runtime_name:"import-ship-transition"`
	StartRectifyFn              func(context.Context) error `runtime:"startup-background-service" runtime_name:"rectify"`
	StartStartramSyncFn         func(context.Context) error `runtime:"startup-background-service,startup-background-service-startram" runtime_name:"startram-sync"`
	StartStartramRenewalFn      func(context.Context) error `runtime:"startup-background-service,startup-background-service-startram" runtime_name:"startram-renewal"`
	StartDockerLogStreamerFn    func(context.Context) error `runtime:"startup-background-service" runtime_name:"docker-log-streamer"`
	StartDockerLogConnRemoverFn func(context.Context) error `runtime:"startup-background-service" runtime_name:"docker-log-conn-remover"`
}

type startBackgroundServicesRuntimeContract interface {
	startupServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec
	validate(startramWgRegistered bool) error
}

type startupBackgroundServiceName string

const (
	startBackgroundServiceVersion              startupBackgroundServiceName = "version"
	startBackgroundServiceDocker               startupBackgroundServiceName = "docker"
	startBackgroundServiceLeak                 startupBackgroundServiceName = "leak"
	startBackgroundServiceSysLogStreamer       startupBackgroundServiceName = "sys-log-streamer"
	startBackgroundServiceOldLogsCleaner       startupBackgroundServiceName = "old-logs-cleaner"
	startBackgroundServiceDiskUsageWarning     startupBackgroundServiceName = "disk-usage-warning"
	startBackgroundServiceSmartDiskCheck       startupBackgroundServiceName = "smart-disk-check"
	startBackgroundServicePackSchedule         startupBackgroundServiceName = "pack-schedule"
	startBackgroundServiceChopRoutines         startupBackgroundServiceName = "chop-routines"
	startBackgroundServiceBackupRoutines       startupBackgroundServiceName = "backup-routines"
	startBackgroundServiceUrbitTransition      startupBackgroundServiceName = "urbit-transition"
	startBackgroundServiceSystemTransition     startupBackgroundServiceName = "system-transition"
	startBackgroundServiceNewShipTransition    startupBackgroundServiceName = "new-ship-transition"
	startBackgroundServiceImportShipTransition startupBackgroundServiceName = "import-ship-transition"
	startBackgroundServiceRectify              startupBackgroundServiceName = "rectify"
	startBackgroundServiceStartramSync         startupBackgroundServiceName = "startram-sync"
	startBackgroundServiceStartramRenewal      startupBackgroundServiceName = "startram-renewal"
	startBackgroundServiceDockerLogStreamer    startupBackgroundServiceName = "docker-log-streamer"
	startBackgroundServiceDockerLogConnRemover startupBackgroundServiceName = "docker-log-conn-remover"
)

type startBackgroundServiceSpecDefinition struct {
	name       startupBackgroundServiceName
	failFast   bool
	requiresWg bool
}

var startBackgroundServiceSpecs = []startBackgroundServiceSpecDefinition{
	{name: startBackgroundServiceVersion, failFast: false},
	{name: startBackgroundServiceDocker, failFast: false},
	{name: startBackgroundServiceLeak, failFast: false},
	{name: startBackgroundServiceSysLogStreamer, failFast: false},
	{name: startBackgroundServiceOldLogsCleaner, failFast: false},
	{name: startBackgroundServiceDiskUsageWarning, failFast: false},
	{name: startBackgroundServiceSmartDiskCheck, failFast: false},
	{name: startBackgroundServicePackSchedule, failFast: false},
	{name: startBackgroundServiceChopRoutines, failFast: false},
	{name: startBackgroundServiceBackupRoutines, failFast: false},
	{name: startBackgroundServiceUrbitTransition, failFast: false},
	{name: startBackgroundServiceSystemTransition, failFast: false},
	{name: startBackgroundServiceNewShipTransition, failFast: false},
	{name: startBackgroundServiceImportShipTransition, failFast: false},
	{name: startBackgroundServiceRectify, failFast: false},
	{name: startBackgroundServiceStartramSync, failFast: true, requiresWg: true},
	{name: startBackgroundServiceStartramRenewal, failFast: false, requiresWg: true},
	{name: startBackgroundServiceDockerLogStreamer, failFast: false},
	{name: startBackgroundServiceDockerLogConnRemover, failFast: false},
}

func startupServiceSpecsForMode(startramWgRegistered bool) []startBackgroundServiceSpecDefinition {
	specs := make([]startBackgroundServiceSpecDefinition, 0, len(startBackgroundServiceSpecs))
	for _, serviceSpec := range startBackgroundServiceSpecs {
		if serviceSpec.requiresWg && !startramWgRegistered {
			continue
		}
		specs = append(specs, serviceSpec)
	}
	return specs
}

func (runtime StartupRuntime) validate() error {
	initErr := runtime.startupInitRuntime.validate()
	runtimeErr := seams.NewCallbackRequirementsWithGroups("startup-runtime").ValidateCallbacks(runtime, "startup runtime")
	if initErr == nil && runtimeErr == nil {
		return nil
	}
	parts := make([]string, 0, 2)
	for _, err := range []error{initErr, runtimeErr} {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return fmt.Errorf("startup runtime missing required callbacks: %s", strings.Join(parts, ", "))
}

type backgroundServiceSpec struct {
	name     string
	startFn  func(context.Context) error
	failFast bool
}

func syncRetrieveWithStartram() error {
	_, err := startram.SyncRetrieve()
	return err
}

type startupTaskID string

func (runtime startBackgroundServicesRuntime) validate(startramWgRegistered bool) error {
	requiredGroups := []string{"startup-background-service"}
	if startramWgRegistered {
		requiredGroups = append(requiredGroups, "startup-background-service-startram")
	}
	if err := seams.NewCallbackRequirementsWithGroups(requiredGroups...).ValidateCallbacks(runtime, "start background services runtime"); err != nil {
		return err
	}
	return nil
}

func (runtime startBackgroundServicesRuntime) startupServiceSpecs(startramWgRegistered bool) []backgroundServiceSpec {
	specs := make([]backgroundServiceSpec, 0, len(startBackgroundServiceSpecs))
	for _, serviceSpec := range startupServiceSpecsForMode(startramWgRegistered) {
		specs = append(specs, backgroundServiceSpec{
			name:     string(serviceSpec.name),
			startFn:  runtime.startServiceFn(serviceSpec.name),
			failFast: serviceSpec.failFast,
		})
	}
	return specs
}

func (runtime startBackgroundServicesRuntime) startServiceFn(name startupBackgroundServiceName) func(context.Context) error {
	switch name {
	case startBackgroundServiceVersion:
		return runtime.StartVersionFn
	case startBackgroundServiceDocker:
		return runtime.StartDockerFn
	case startBackgroundServiceLeak:
		return runtime.StartLeakFn
	case startBackgroundServiceSysLogStreamer:
		return runtime.StartSysLogStreamerFn
	case startBackgroundServiceOldLogsCleaner:
		return runtime.StartOldLogsCleanerFn
	case startBackgroundServiceDiskUsageWarning:
		return runtime.StartDiskUsageWarningFn
	case startBackgroundServiceSmartDiskCheck:
		return runtime.StartSmartDiskCheckFn
	case startBackgroundServicePackSchedule:
		return runtime.StartPackScheduleFn
	case startBackgroundServiceChopRoutines:
		return runtime.StartChopRoutinesFn
	case startBackgroundServiceBackupRoutines:
		return runtime.StartBackupRoutinesFn
	case startBackgroundServiceUrbitTransition:
		return runtime.StartUrbitTransitionFn
	case startBackgroundServiceSystemTransition:
		return runtime.StartSystemTransitionFn
	case startBackgroundServiceNewShipTransition:
		return runtime.StartNewShipTransitionFn
	case startBackgroundServiceImportShipTransition:
		return runtime.StartImportShipTransitionFn
	case startBackgroundServiceRectify:
		return runtime.StartRectifyFn
	case startBackgroundServiceStartramSync:
		return runtime.StartStartramSyncFn
	case startBackgroundServiceStartramRenewal:
		return runtime.StartStartramRenewalFn
	case startBackgroundServiceDockerLogStreamer:
		return runtime.StartDockerLogStreamerFn
	case startBackgroundServiceDockerLogConnRemover:
		return runtime.StartDockerLogConnRemoverFn
	default:
		return nil
	}
}

func resolveStartupRuntime(runtime StartupRuntime) (StartupRuntime, error) {
	resolvedRuntime := startupRuntimeWithDefaults(runtime)
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
	resolved := startBackgroundServicesRuntimeWithDefaults(logstreamRuntime, runtime)
	if err := resolved.validate(startramWgRegistered); err != nil {
		return resolved, err
	}
	return resolved, nil
}

func startupInitRuntimeWithDefaults(overrides startupInitRuntime) startupInitRuntime {
	runtime := startupInitDefaultRuntime()
	if overrides.InitializeConfigFn != nil {
		runtime.InitializeConfigFn = overrides.InitializeConfigFn
	}
	if overrides.InitializeAuthFn != nil {
		runtime.InitializeAuthFn = overrides.InitializeAuthFn
	}
	if overrides.InitializeRouterFn != nil {
		runtime.InitializeRouterFn = overrides.InitializeRouterFn
	}
	if overrides.InitializeSystemSupportFn != nil {
		runtime.InitializeSystemSupportFn = overrides.InitializeSystemSupportFn
	}
	if overrides.InitializeExporterFn != nil {
		runtime.InitializeExporterFn = overrides.InitializeExporterFn
	}
	if overrides.InitializeImporterFn != nil {
		runtime.InitializeImporterFn = overrides.InitializeImporterFn
	}
	if overrides.InitializeBroadcastFn != nil {
		runtime.InitializeBroadcastFn = overrides.InitializeBroadcastFn
	}
	if overrides.InitializeDockerFn != nil {
		runtime.InitializeDockerFn = overrides.InitializeDockerFn
	}
	if overrides.InitializeWiFiFn != nil {
		runtime.InitializeWiFiFn = overrides.InitializeWiFiFn
	}
	if overrides.StartMDNSServerFn != nil {
		runtime.StartMDNSServerFn = overrides.StartMDNSServerFn
	}
	if overrides.InitializeResolvedFn != nil {
		runtime.InitializeResolvedFn = overrides.InitializeResolvedFn
	}
	if overrides.NetworkReachabilityFn != nil {
		runtime.NetworkReachabilityFn = overrides.NetworkReachabilityFn
	}
	if overrides.PrimeRekorKeyFn != nil {
		runtime.PrimeRekorKeyFn = overrides.PrimeRekorKeyFn
	}
	if overrides.ConfigureSwapFn != nil {
		runtime.ConfigureSwapFn = overrides.ConfigureSwapFn
	}
	if overrides.SwapSettingsFn != nil {
		runtime.SwapSettingsFn = overrides.SwapSettingsFn
	}
	if overrides.SetupTmpDirFn != nil {
		runtime.SetupTmpDirFn = overrides.SetupTmpDirFn
	}
	return runtime
}

func startupRuntimeWithDefaults(overrides StartupRuntime) StartupRuntime {
	base := defaultStartupRuntime()
	base.startupInitRuntime = startupInitRuntimeWithDefaults(overrides.startupInitRuntime)
	if overrides.StartConfigEventLoopFn != nil {
		base.StartConfigEventLoopFn = overrides.StartConfigEventLoopFn
	}
	if overrides.StartStartupContainersFn != nil {
		base.StartStartupContainersFn = overrides.StartStartupContainersFn
	}
	if overrides.StartupUpdateSettingsFn != nil {
		base.StartupUpdateSettingsFn = overrides.StartupUpdateSettingsFn
	}
	if overrides.StartupStartramSettingsFn != nil {
		base.StartupStartramSettingsFn = overrides.StartupStartramSettingsFn
	}
	return base
}

func startBackgroundServicesRuntimeWithDefaults(logstreamRuntime *logstream.LogstreamRuntime, overrides startBackgroundServicesRuntime) startBackgroundServicesRuntime {
	base := defaultStartBackgroundServicesRuntime(logstreamRuntime)
	if overrides.StartVersionFn != nil {
		base.StartVersionFn = overrides.StartVersionFn
	}
	if overrides.StartDockerFn != nil {
		base.StartDockerFn = overrides.StartDockerFn
	}
	if overrides.StartLeakFn != nil {
		base.StartLeakFn = overrides.StartLeakFn
	}
	if overrides.StartSysLogStreamerFn != nil {
		base.StartSysLogStreamerFn = overrides.StartSysLogStreamerFn
	}
	if overrides.StartOldLogsCleanerFn != nil {
		base.StartOldLogsCleanerFn = overrides.StartOldLogsCleanerFn
	}
	if overrides.StartDiskUsageWarningFn != nil {
		base.StartDiskUsageWarningFn = overrides.StartDiskUsageWarningFn
	}
	if overrides.StartSmartDiskCheckFn != nil {
		base.StartSmartDiskCheckFn = overrides.StartSmartDiskCheckFn
	}
	if overrides.StartPackScheduleFn != nil {
		base.StartPackScheduleFn = overrides.StartPackScheduleFn
	}
	if overrides.StartChopRoutinesFn != nil {
		base.StartChopRoutinesFn = overrides.StartChopRoutinesFn
	}
	if overrides.StartBackupRoutinesFn != nil {
		base.StartBackupRoutinesFn = overrides.StartBackupRoutinesFn
	}
	if overrides.StartUrbitTransitionFn != nil {
		base.StartUrbitTransitionFn = overrides.StartUrbitTransitionFn
	}
	if overrides.StartSystemTransitionFn != nil {
		base.StartSystemTransitionFn = overrides.StartSystemTransitionFn
	}
	if overrides.StartNewShipTransitionFn != nil {
		base.StartNewShipTransitionFn = overrides.StartNewShipTransitionFn
	}
	if overrides.StartImportShipTransitionFn != nil {
		base.StartImportShipTransitionFn = overrides.StartImportShipTransitionFn
	}
	if overrides.StartRectifyFn != nil {
		base.StartRectifyFn = overrides.StartRectifyFn
	}
	if overrides.StartStartramSyncFn != nil {
		base.StartStartramSyncFn = overrides.StartStartramSyncFn
	}
	if overrides.StartStartramRenewalFn != nil {
		base.StartStartramRenewalFn = overrides.StartStartramRenewalFn
	}
	if overrides.StartDockerLogStreamerFn != nil {
		base.StartDockerLogStreamerFn = overrides.StartDockerLogStreamerFn
	}
	if overrides.StartDockerLogConnRemoverFn != nil {
		base.StartDockerLogConnRemoverFn = overrides.StartDockerLogConnRemoverFn
	}
	return base
}

func defaultStartBackgroundServicesRuntime(logstreamRuntime *logstream.LogstreamRuntime) startBackgroundServicesRuntime {
	return startBackgroundServicesRuntime{
		StartVersionFn:          routines.StartVersionSubsystemWithContext,
		StartDockerFn:           subsystem.StartDockerSubsystem,
		StartLeakFn:             leak.StartLeakWithContext,
		StartSysLogStreamerFn:   func(ctx context.Context) error { return logstream.SysLogStreamerWithRuntime(ctx, logstreamRuntime) },
		StartOldLogsCleanerFn:   logstream.OldLogsCleanerWithContext,
		StartDiskUsageWarningFn: routines.DiskUsageWarningWithContext,
		StartSmartDiskCheckFn:   routines.SmartDiskCheckWithContext,
		StartPackScheduleFn:     routines.PackScheduleLoopWithContext,
		StartChopRoutinesFn:     routines.StartChopRoutinesWithContext,
		StartBackupRoutinesFn:   routines.StartBackupRoutinesWithContext,
		StartUrbitTransitionFn: func(ctx context.Context) error {
			return rectify.UrbitTransitionHandlerWithContextAndRuntime(ctx, rectify.NewRectifyRuntime())
		},
		StartSystemTransitionFn: func(ctx context.Context) error {
			return rectify.SystemTransitionHandlerWithContextAndRuntime(ctx, rectify.NewRectifyRuntime())
		},
		StartNewShipTransitionFn: func(ctx context.Context) error {
			return rectify.NewShipTransitionHandlerWithContextAndRuntime(ctx, rectify.NewRectifyRuntime())
		},
		StartImportShipTransitionFn: func(ctx context.Context) error {
			return rectify.ImportShipTransitionHandlerWithContextAndRuntime(ctx, nil)
		},
		StartRectifyFn:           rectify.RectifyUrbitWithContext,
		StartStartramSyncFn:      func(context.Context) error { return syncRetrieveWithStartram() },
		StartStartramRenewalFn:   routines.StartramRenewalReminderWithContext,
		StartDockerLogStreamerFn: func(ctx context.Context) error { return logstream.DockerLogStreamerWithRuntime(ctx, logstreamRuntime) },
		StartDockerLogConnRemoverFn: func(ctx context.Context) error {
			return logstream.DockerLogConnRemoverWithRuntime(ctx, logstreamRuntime)
		},
	}
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
				updateSettings := opts.StartupRuntime.StartupUpdateSettingsFn()
				if opts.StartupRuntime.StartConfigEventLoopFn == nil {
					return fmt.Errorf("startup runtime is missing startConfigEventLoop callback")
				}
				if err := opts.StartupRuntime.StartConfigEventLoopFn(ctx); err != nil {
					return fmt.Errorf("start config event loop failed: %w", err)
				}
				if opts.StartupRuntime.StartupStartramSettingsFn == nil {
					return fmt.Errorf("startup runtime is missing startupStartramSettings callback")
				}
				startramSettings := opts.StartupRuntime.StartupStartramSettingsFn()
				versionUpdateChannel, remoteVersion := startVersionDiscovery(updateSettings.UpdateMode, updateSettings.UpdateBranch)
				services, err := startBackgroundServices(ctx, startramSettings.WgRegistered, opts.startC2cCheck)
				if err != nil {
					return err
				}
				backgroundServices = services

				waitForVersionDiscovery(remoteVersion, versionUpdateChannel, updateSettings.UpdateBranch)
				if opts.StartupRuntime.StartStartupContainersFn == nil {
					return fmt.Errorf("startup runtime is missing startStartupContainers callback")
				}
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

func startBackgroundServicesWithRuntime(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context) error, runtime startBackgroundServicesRuntimeContract) (*startupBackgroundServices, error) {
	services, specs, err := buildBackgroundServices(ctx, startramWgRegistered, startC2cCheck, runtime)
	if err != nil {
		return nil, err
	}
	for _, service := range specs {
		handle := superviseBackgroundService(ctx, service.name, service.startFn)
		services.add(handle)
		if !service.failFast {
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

func buildBackgroundServices(ctx context.Context, startramWgRegistered bool, startC2cCheck func(context.Context) error, runtime startBackgroundServicesRuntimeContract) (*startupBackgroundServices, []backgroundServiceSpec, error) {
	var specs []backgroundServiceSpec
	switch typedRuntime := runtime.(type) {
	case startBackgroundServicesRuntime:
		resolvedRuntime, err := resolveStartBackgroundServicesRuntime(logstreamRuntimeFromContext(), startramWgRegistered, typedRuntime)
		if err != nil {
			return nil, nil, err
		}
		specs = resolvedRuntime.startupServiceSpecs(startramWgRegistered)
	case *startBackgroundServicesRuntime:
		normalized := startBackgroundServicesRuntime{}
		if typedRuntime != nil {
			normalized = *typedRuntime
		}
		resolvedRuntime, err := resolveStartBackgroundServicesRuntime(logstreamRuntimeFromContext(), startramWgRegistered, normalized)
		if err != nil {
			return nil, nil, err
		}
		specs = resolvedRuntime.startupServiceSpecs(startramWgRegistered)
	default:
		if err := runtime.validate(startramWgRegistered); err != nil {
			return nil, nil, err
		}
		specs = runtime.startupServiceSpecs(startramWgRegistered)
	}

	services := &startupBackgroundServices{}
	if startC2cCheck != nil {
		services.add(superviseBackgroundService(ctx, "c2c-check", func(ctx context.Context) error {
			return startC2cCheck(ctx)
		}))
	}
	return services, specs, nil
}

func logstreamRuntimeFromContext() *logstream.LogstreamRuntime {
	systemRuntime := session.LogstreamRuntimeState()
	logger.ConfigureLogstreamRuntime(systemRuntime)
	return logstream.Configure(systemRuntime, systemRuntime.SystemLogMessages())
}

func loadService(loadFn func() error, failureMessage string) error {
	if loadFn == nil {
		return fmt.Errorf("startup load function is not configured: %s", failureMessage)
	}
	if err := loadFn(); err != nil {
		return fmt.Errorf("%s: %w", failureMessage, err)
	}
	return nil
}
