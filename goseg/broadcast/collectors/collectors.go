package collectors

import (
	"errors"
	"fmt"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"groundseg/transition"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

var startramServicesRetriever = startram.Retrieve

var (
	errCollectorStartramSettingsMissing = errors.New("collector runtime startram settings callback is not configured")
	errCollectorStartramConfigMissing   = errors.New("collector runtime startram config callback is not configured")
	errCollectorPenpaiSettingsMissing   = errors.New("collector runtime penpai settings callback is not configured")
	errCollectorSwapSettingsMissing     = errors.New("collector runtime swap settings callback is not configured")
	errCollectorBackupRootMissing       = errors.New("collector runtime backup root callback is not configured")
	errCollectorBackupTimeMissing       = errors.New("collector runtime backup time callback is not configured")
)

type collectorRuntime struct {
	loadUrbitConfigFn        func(string) error
	urbitConfFn              func(string) structs.UrbitDocker
	getContainerStatsFn      func(string) structs.ContainerStats
	getContainerImageTagFn   func(string) (string, error)
	getMinIOLinkedStatusFn   func(string) bool
	getMinIOPasswordFn       func(string) (string, error)
	getContainerNetworkFn    func(string) (string, error)
	getContainerShipStatusFn func([]string) (map[string]string, error)
	lusCodeFn                func(string) (string, error)
	getDeskFn                func(string, string, bool) (string, error)
	startramSettingsFn       func() config.StartramSettings
	startramConfigFn         func() structs.StartramRetrieve
	penpaiSettingsFn         func() config.PenpaiSettings
	swapSettingsFn           func() config.SwapSettings
	backupRootFn             func() string
	backupTimeFn             func() time.Time
}

func defaultCollectorRuntime() collectorRuntime {
	networkRuntime := network.NewNetworkRuntime()
	return collectorRuntime{
		loadUrbitConfigFn:        config.LoadUrbitConfig,
		urbitConfFn:              config.UrbitConf,
		getContainerStatsFn:      orchestration.GetContainerStats,
		getContainerImageTagFn:   orchestration.GetContainerImageTag,
		getMinIOLinkedStatusFn:   config.GetMinIOLinkedStatus,
		getMinIOPasswordFn:       config.GetMinIOPassword,
		getContainerNetworkFn:    networkRuntime.GetContainerNetwork,
		getContainerShipStatusFn: orchestration.GetShipStatus,
		lusCodeFn:                click.GetLusCode,
		getDeskFn:                click.GetDesk,
		startramSettingsFn:       config.StartramSettingsSnapshot,
		startramConfigFn:         config.GetStartramConfig,
		penpaiSettingsFn:         config.PenpaiSettingsSnapshot,
		swapSettingsFn:           config.SwapSettingsSnapshot,
		backupRootFn:             func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) },
		backupTimeFn:             func() time.Time { return config.BackupTime },
	}
}

// ConstructPierInfo builds the urbit entries for broadcast state.
func ConstructPierInfo(existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return constructPierInfo(defaultCollectorRuntime(), existingUrbits, scheduled)
}

func constructPierInfo(runtime collectorRuntime, existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	if err := runtime.ensureConfigured(); err != nil {
		return nil, err
	}
	settings := runtime.startramSettings()
	piers := settings.Piers
	sgContext := wireguardContext{
		registered: settings.WgRegistered,
		on:         settings.WgOn,
	}

	backups := backupSnapshotForPiersForRuntime(runtime, piers, runtime.startramConfig().Backups)
	rtSnapshot, err := runtimeSnapshotForPiersForRuntime(runtime, piers, existingUrbits)
	if err != nil {
		return nil, fmt.Errorf("constructing pier info: %w", err)
	}
	startramSnapshot := startramSnapshotForPiers(runtime.startramConfig().Subdomains)
	deploymentInputs := collectUrbitDeploymentInputsForPiersForRuntime(runtime, piers, rtSnapshot.hostName, sgContext, startramSnapshot)
	runtimeInputs := collectUrbitRuntimeInputsForPiersForRuntime(
		runtime,
		rtSnapshot.pierStatus,
		urbitRuntimeContext{
			existingUrbits: rtSnapshot.currentState,
			shipNetworks:   rtSnapshot.shipNetworks,
		},
		scheduled,
	)
	return composeUrbitViewsForRuntime(piers, runtimeInputs, deploymentInputs, backups), nil
}

func ConstructAppsInfo() structs.Apps {
	runtime := defaultCollectorRuntime()
	if err := runtime.ensureConfigured(); err != nil {
		panic(err)
	}
	return appInfoCollector{}.collect(runtime)
}

func ConstructProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	runtime := defaultCollectorRuntime()
	if err := runtime.ensureConfigured(); err != nil {
		panic(err)
	}
	return profileInfoCollector{}.collect(runtime, regions)
}

// ConstructSystemInfo builds system usage and update health info.
func ConstructSystemInfo() structs.System {
	runtime := defaultCollectorRuntime()
	if err := runtime.ensureConfigured(); err != nil {
		panic(err)
	}
	return systemInfoCollector{}.collect(runtime)
}

func (runtime collectorRuntime) ensureConfigured() error {
	if runtime.startramSettingsFn == nil {
		return errCollectorStartramSettingsMissing
	}
	if runtime.startramConfigFn == nil {
		return errCollectorStartramConfigMissing
	}
	if runtime.penpaiSettingsFn == nil {
		return errCollectorPenpaiSettingsMissing
	}
	if runtime.swapSettingsFn == nil {
		return errCollectorSwapSettingsMissing
	}
	if runtime.backupRootFn == nil {
		return errCollectorBackupRootMissing
	}
	if runtime.backupTimeFn == nil {
		return errCollectorBackupTimeMissing
	}
	return nil
}

func LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	regions, err := startram.SyncRegions()
	if err != nil {
		return nil, fmt.Errorf("loading startram regions: %w", err)
	}
	return regions, nil
}

func GetStartramServices() error {
	zap.L().Info("Retrieving StarTram services info")
	res, err := startramServicesRetriever()
	if err != nil {
		return fmt.Errorf("retrieve startram services: %w", err)
	}
	zap.L().Info(fmt.Sprintf("%+v", res.Subdomains))
	return nil
}

type pierBackupSnapshot struct {
	remote       structs.Backup
	localDaily   structs.Backup
	localWeekly  structs.Backup
	localMonthly structs.Backup
}

type pierRuntimeSnapshot struct {
	currentState map[string]structs.Urbit
	shipNetworks map[string]string
	pierStatus   map[string]string
	hostName     string
}

type pierStartramSnapshot struct {
	remoteReadyByURL map[string]bool
}

func backupSnapshotForPiers(piers []string, remoteBackups []structs.Backup) pierBackupSnapshot {
	return backupSnapshotForPiersForRuntime(defaultCollectorRuntime(), piers, remoteBackups)
}

func backupSnapshotForPiersForRuntime(runtime collectorRuntime, piers []string, remoteBackups []structs.Backup) pierBackupSnapshot {
	return pierBackupSnapshot{
		remote:       flattenRemoteBackups(remoteBackups),
		localDaily:   localBackupsForPeriodForRuntime(runtime, piers, "daily"),
		localWeekly:  localBackupsForPeriodForRuntime(runtime, piers, "weekly"),
		localMonthly: localBackupsForPeriodForRuntime(runtime, piers, "monthly"),
	}
}

func flattenRemoteBackups(remoteBackups []structs.Backup) structs.Backup {
	remoteBackupMap := make(structs.Backup)
	for _, backup := range remoteBackups {
		for ship, backupInfo := range backup {
			remoteBackupMap[ship] = backupInfo
		}
	}
	return remoteBackupMap
}

func localBackupsForPeriod(piers []string, period string) structs.Backup {
	return localBackupsForPeriodForRuntime(defaultCollectorRuntime(), piers, period)
}

func localBackupsForPeriodForRuntime(runtime collectorRuntime, piers []string, period string) structs.Backup {
	localBackups := make(structs.Backup)
	backupDir := runtime.backupRoot()
	for _, ship := range piers {
		if backupDir == "" {
			continue
		}
		shipBackups, err := filepath.Glob(filepath.Join(backupDir, ship, period, "*"))
		if err != nil {
			continue
		}
		for _, backup := range shipBackups {
			timestamp, err := strconv.Atoi(filepath.Base(backup))
			if err != nil {
				continue
			}
			localBackups[ship] = append(localBackups[ship], structs.BackupObject{Timestamp: timestamp, MD5: ""})
		}
	}
	return localBackups
}

func (runtime collectorRuntime) startramSettings() config.StartramSettings {
	if runtime.startramSettingsFn == nil {
		panic(errCollectorStartramSettingsMissing)
	}
	return runtime.startramSettingsFn()
}

func (runtime collectorRuntime) startramConfig() structs.StartramRetrieve {
	if runtime.startramConfigFn == nil {
		panic(errCollectorStartramConfigMissing)
	}
	return runtime.startramConfigFn()
}

func (runtime collectorRuntime) penpaiSettings() config.PenpaiSettings {
	if runtime.penpaiSettingsFn == nil {
		panic(errCollectorPenpaiSettingsMissing)
	}
	return runtime.penpaiSettingsFn()
}

func (runtime collectorRuntime) swapSettings() config.SwapSettings {
	if runtime.swapSettingsFn == nil {
		panic(errCollectorSwapSettingsMissing)
	}
	return runtime.swapSettingsFn()
}

func (runtime collectorRuntime) backupRoot() string {
	if runtime.backupRootFn == nil {
		panic(errCollectorBackupRootMissing)
	}
	return runtime.backupRootFn()
}

func (runtime collectorRuntime) backupTime() time.Time {
	if runtime.backupTimeFn == nil {
		panic(errCollectorBackupTimeMissing)
	}
	return runtime.backupTimeFn()
}

func runtimeSnapshotForPiers(piers []string, urbits map[string]structs.Urbit) (pierRuntimeSnapshot, error) {
	return runtimeSnapshotForPiersForRuntime(defaultCollectorRuntime(), piers, urbits)
}

func runtimeSnapshotForPiersForRuntime(runtime collectorRuntime, piers []string, urbits map[string]structs.Urbit) (pierRuntimeSnapshot, error) {
	snapshot := pierRuntimeSnapshot{
		currentState: urbits,
		shipNetworks: getContainerNetworksWithLookup(runtime.getContainerNetworkFn, piers),
		hostName:     resolveBroadcastHostName(),
	}
	pierStatus, err := runtime.getContainerShipStatusFn(piers)
	if err != nil {
		return snapshot, fmt.Errorf("getting container ship status: %w", err)
	}
	snapshot.pierStatus = pierStatus
	return snapshot, nil
}

func resolveBroadcastHostName() string {
	hostName := system.LocalUrl
	if hostName == "" {
		zap.L().Debug("Defaulting to `nativeplanet.local`")
		hostName = "nativeplanet.local"
	}
	return hostName
}

func startramSnapshotForPiers(subdomains []structs.Subdomain) pierStartramSnapshot {
	readyByURL := make(map[string]bool, len(subdomains))
	for _, subdomain := range subdomains {
		readyByURL[subdomain.URL] = subdomain.Status == string(transition.StartramServiceStatusOk)
	}
	return pierStartramSnapshot{
		remoteReadyByURL: readyByURL,
	}
}

func normalizePackSchedule(meldDay string, meldDate int) (string, int) {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	packDay := "Monday"
	for _, day := range days {
		if day == meldDay {
			packDay = strings.Title(meldDay)
			break
		}
	}
	packDate := 1
	if meldDate > 1 {
		packDate = meldDate
	}
	return packDay, packDate
}

func collectUrbitDeploymentInputsForRuntime(
	runtime collectorRuntime,
	pier string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) (urbitDeploymentInputs, bool) {
	if err := runtime.loadUrbitConfigFn(pier); err != nil {
		errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
		zap.L().Error(errmsg)
		return urbitDeploymentInputs{}, false
	}

	dockerConfig := runtime.urbitConfFn(pier)
	remote := dockerConfig.Network == "wireguard"
	url := fmt.Sprintf("http://%s:%d", hostName, dockerConfig.HTTPPort)
	if remote {
		url = fmt.Sprintf("https://%s", dockerConfig.WgURL)
	}

	minIOPwd := dockerConfig.MinioPassword
	if wireguardCtx.registered && wireguardCtx.on {
		retrievedPwd, err := runtime.getMinIOPasswordFn(fmt.Sprintf("minio_%s", pier))
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to refresh MinIO password for %s: %v", pier, err))
		} else {
			minIOPwd = retrievedPwd
		}
	}
	packDay, packDate := normalizePackSchedule(dockerConfig.MeldDay, dockerConfig.MeldDate)
	minIOLinked := runtime.getMinIOLinkedStatusFn(pier)

	return urbitDeploymentInputs{
		dockerConfig:        dockerConfig,
		url:                 url,
		remote:              remote,
		remoteReady:         startramSnapshot.remoteReadyByURL[dockerConfig.WgURL],
		showUrbitWebAlias:   dockerConfig.ShowUrbitWeb == "custom",
		minIOPwd:            minIOPwd,
		disableShipRestarts: !dockerConfig.DisableShipRestarts,
		bootStatus:          dockerConfig.BootStatus != "ignore",
		minIOLinked:         minIOLinked,
		startramReminder:    dockerConfig.StartramReminder,
		chopOnUpgrade:       dockerConfig.ChopOnUpgrade,
		packDay:             packDay,
		packDate:            packDate,
		minIOURL:            fmt.Sprintf("https://console.s3.%s", dockerConfig.WgURL),
	}, true
}

func collectUrbitDeploymentInputs(
	pier string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) (urbitDeploymentInputs, bool) {
	return collectUrbitDeploymentInputsForRuntime(defaultCollectorRuntime(), pier, hostName, wireguardCtx, startramSnapshot)
}

func collectUrbitDeploymentInputsForPiers(
	piers []string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) map[string]urbitDeploymentInputs {
	return collectUrbitDeploymentInputsForPiersForRuntime(
		defaultCollectorRuntime(),
		piers,
		hostName,
		wireguardCtx,
		startramSnapshot,
	)
}

func collectUrbitDeploymentInputsForPiersForRuntime(
	runtime collectorRuntime,
	piers []string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) map[string]urbitDeploymentInputs {
	inputs := make(map[string]urbitDeploymentInputs, len(piers))
	for _, pier := range piers {
		deploymentInputs, ok := collectUrbitDeploymentInputsForRuntime(runtime, pier, hostName, wireguardCtx, startramSnapshot)
		if !ok {
			continue
		}
		inputs[pier] = deploymentInputs
	}
	return inputs
}

type urbitRuntimeContext struct {
	existingUrbits map[string]structs.Urbit
	shipNetworks   map[string]string
}

func collectUrbitRuntimeInputsForRuntime(
	runtime collectorRuntime,
	pier, status string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
) urbitRuntimeInputs {
	existing := structs.Urbit{}
	if existingUrbit, exists := rtContext.existingUrbits[pier]; exists {
		existing = existingUrbit
	}
	isRunning := status == "Up" || strings.HasPrefix(status, "Up ")
	lusCode, err := lusCodeForRuntime(runtime, pier, status)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to resolve +code for %s: %v", pier, err))
	}
	packUnixTime := int64(0)
	if getScheduledPack != nil {
		packUnixTime = getScheduledPack(pier).Unix()
	}
	return urbitRuntimeInputs{
		existingUrbit:     existing,
		containerImageTag: func() (string, error) { return runtime.getContainerImageTagFn(pier) },
		dockerStats:       runtime.getContainerStatsFn(pier),
		network:           rtContext.shipNetworks[pier],
		isRunning:         isRunning,
		lusCode:           lusCode,
		penpaiCompanion:   deskInstalledForRuntime(runtime, pier, status, "penpai"),
		gallsegInstalled:  deskInstalledForRuntime(runtime, pier, status, "groundseg"),
		packUnixTime:      packUnixTime,
	}
}

func collectUrbitRuntimeInputs(
	pier, status string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
) urbitRuntimeInputs {
	return collectUrbitRuntimeInputsForRuntime(defaultCollectorRuntime(), pier, status, rtContext, getScheduledPack)
}

func collectUrbitRuntimeInputsForPiers(
	pierStatus map[string]string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
) map[string]urbitRuntimeInputs {
	inputs := make(map[string]urbitRuntimeInputs, len(pierStatus))
	for pier, status := range pierStatus {
		inputs[pier] = collectUrbitRuntimeInputsForRuntime(defaultCollectorRuntime(), pier, status, rtContext, getScheduledPack)
	}
	return inputs
}

func collectUrbitRuntimeInputsForPiersForRuntime(
	runtime collectorRuntime,
	pierStatus map[string]string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
) map[string]urbitRuntimeInputs {
	inputs := make(map[string]urbitRuntimeInputs, len(pierStatus))
	for pier, status := range pierStatus {
		inputs[pier] = collectUrbitRuntimeInputsForRuntime(runtime, pier, status, rtContext, getScheduledPack)
	}
	return inputs
}

func composeUrbitViewInputs(runtimeInputs urbitRuntimeInputs, deployInputs urbitDeploymentInputs) urbitViewInputs {
	runningTag := ""
	if runtimeInputs.containerImageTag != nil {
		if tag, err := runtimeInputs.containerImageTag(); err == nil && tag != "" {
			runningTag = tag
		}
	}
	return urbitViewInputs{
		existingUrbit:       runtimeInputs.existingUrbit,
		dockerConfig:        deployInputs.dockerConfig,
		dockerStats:         runtimeInputs.dockerStats,
		network:             runtimeInputs.network,
		isRunning:           runtimeInputs.isRunning,
		minIOLinked:         deployInputs.minIOLinked,
		bootStatus:          deployInputs.bootStatus,
		remote:              deployInputs.remote,
		url:                 deployInputs.url,
		remoteReady:         deployInputs.remoteReady,
		showUrbitWebAlias:   deployInputs.showUrbitWebAlias,
		minIOPwd:            deployInputs.minIOPwd,
		disableShipRestarts: deployInputs.disableShipRestarts,
		lusCode:             runtimeInputs.lusCode,
		penpaiCompanion:     runtimeInputs.penpaiCompanion,
		gallsegInstalled:    runtimeInputs.gallsegInstalled,
		startramReminder:    deployInputs.startramReminder,
		chopOnUpgrade:       deployInputs.chopOnUpgrade,
		packDay:             deployInputs.packDay,
		packDate:            deployInputs.packDate,
		minIOURL:            deployInputs.minIOURL,
		packUnixTime:        runtimeInputs.packUnixTime,
		containerImageTag:   runningTag,
	}
}

func composeUrbitViews(
	piers []string,
	runtimeInputs map[string]urbitRuntimeInputs,
	deploymentInputs map[string]urbitDeploymentInputs,
	backups pierBackupSnapshot,
) map[string]structs.Urbit {
	return composeUrbitViewsForRuntime(piers, runtimeInputs, deploymentInputs, backups)
}

func composeUrbitViewsForRuntime(
	piers []string,
	runtimeInputs map[string]urbitRuntimeInputs,
	deploymentInputs map[string]urbitDeploymentInputs,
	backups pierBackupSnapshot,
) map[string]structs.Urbit {
	updates := make(map[string]structs.Urbit, len(piers))
	for _, pier := range piers {
		runtimeInput, ok := runtimeInputs[pier]
		if !ok {
			continue
		}
		deployInput, ok := deploymentInputs[pier]
		if !ok {
			continue
		}
		urbit, ok := urbitViewMapper{}.assemble(
			pier,
			composeUrbitViewInputs(runtimeInput, deployInput),
			backups,
		)
		if !ok {
			continue
		}
		updates[pier] = urbit
	}
	return updates
}

func lusCodeForRuntime(runtime collectorRuntime, pier, status string) (string, error) {
	if !transition.IsContainerUpStatus(status) {
		return "", nil
	}
	lusCode, err := runtime.lusCodeFn(pier)
	if err != nil {
		return "", fmt.Errorf("unable to resolve +code for %s: %w", pier, err)
	}
	return lusCode, nil
}

func lusCodeIfRunning(pier string, status string) (string, error) {
	return lusCodeForRuntime(defaultCollectorRuntime(), pier, status)
}

func deskInstalledForRuntime(runtime collectorRuntime, pier, status, desk string) bool {
	if !transition.IsContainerUpStatus(status) {
		return false
	}
	deskStatus, err := runtime.getDeskFn(pier, desk, false)
	if err != nil {
		zap.L().Debug(fmt.Sprintf("Broadcast failed to get %s desk info for %v: %v", desk, pier, err))
		return false
	}
	return deskStatus == string(transition.ContainerStatusRunning)
}

func deskInstalledIfRunning(pier, status, desk string) bool {
	return deskInstalledForRuntime(defaultCollectorRuntime(), pier, status, desk)
}

type urbitDeploymentInputs struct {
	dockerConfig        structs.UrbitDocker
	url                 string
	remote              bool
	remoteReady         bool
	showUrbitWebAlias   bool
	minIOPwd            string
	disableShipRestarts bool
	minIOLinked         bool
	bootStatus          bool
	startramReminder    bool
	chopOnUpgrade       bool
	packDay             string
	packDate            int
	minIOURL            string
}

type wireguardContext struct {
	registered bool
	on         bool
}

type urbitRuntimeInputs struct {
	existingUrbit     structs.Urbit
	dockerStats       structs.ContainerStats
	network           string
	isRunning         bool
	containerImageTag func() (string, error)
	lusCode           string
	penpaiCompanion   bool
	gallsegInstalled  bool
	packUnixTime      int64
}

type urbitViewInputs struct {
	existingUrbit       structs.Urbit
	dockerConfig        structs.UrbitDocker
	dockerStats         structs.ContainerStats
	network             string
	isRunning           bool
	bootStatus          bool
	remote              bool
	url                 string
	remoteReady         bool
	showUrbitWebAlias   bool
	minIOPwd            string
	disableShipRestarts bool
	lusCode             string
	minIOLinked         bool
	containerImageTag   string
	penpaiCompanion     bool
	gallsegInstalled    bool
	startramReminder    bool
	chopOnUpgrade       bool
	packDay             string
	packDate            int
	minIOURL            string
	packUnixTime        int64
}

type urbitViewMapper struct{}

func (urbitViewMapper) assemble(
	pier string,
	inputs urbitViewInputs,
	backups pierBackupSnapshot,
) (structs.Urbit, bool) {
	return assembleUrbitViewFromInputs(pier, inputs, backups), true
}

func assembleUrbitViewFromInputs(pier string, inputs urbitViewInputs, backups pierBackupSnapshot) structs.Urbit {
	urbit := inputs.existingUrbit

	urbit.Info.LusCode = inputs.lusCode
	urbit.Info.Running = inputs.isRunning
	urbit.Info.Network = inputs.network
	urbit.Info.URL = inputs.url
	urbit.Info.LoomSize = inputs.dockerConfig.LoomSize
	urbit.Info.DiskUsage = inputs.dockerStats.DiskUsage
	urbit.Info.MemUsage = inputs.dockerStats.MemoryUsage
	urbit.Info.DevMode = inputs.dockerConfig.DevMode
	urbit.Info.DetectBootStatus = inputs.bootStatus
	urbit.Info.Remote = inputs.remote
	urbit.Info.RemoteReady = inputs.remoteReady
	if runningTag := inputs.containerImageTag; runningTag != "" {
		urbit.Info.Vere = runningTag
	} else {
		urbit.Info.Vere = inputs.dockerConfig.UrbitVersion
	}
	urbit.Info.MinIOUrl = inputs.minIOURL
	urbit.Info.MinIOPwd = inputs.minIOPwd
	urbit.Info.UrbitAlias = inputs.dockerConfig.CustomUrbitWeb
	urbit.Info.MinIOAlias = inputs.dockerConfig.CustomS3Web
	urbit.Info.ShowUrbAlias = inputs.showUrbitWebAlias
	urbit.Info.MinIOLinked = inputs.minIOLinked
	urbit.Info.PackScheduleActive = inputs.dockerConfig.MeldSchedule
	urbit.Info.PackDay = inputs.packDay
	urbit.Info.PackDate = inputs.packDate
	urbit.Info.PackTime = inputs.dockerConfig.MeldTime
	urbit.Info.LastPack = inputs.dockerConfig.MeldLast
	urbit.Info.NextPack = fmt.Sprintf("%d", inputs.packUnixTime)
	urbit.Info.PackIntervalType = inputs.dockerConfig.MeldScheduleType
	urbit.Info.PackIntervalValue = inputs.dockerConfig.MeldFrequency
	urbit.Info.PenpaiCompanion = inputs.penpaiCompanion
	urbit.Info.Gallseg = inputs.gallsegInstalled
	urbit.Info.StartramReminder = inputs.startramReminder
	urbit.Info.ChopOnUpgrade = inputs.chopOnUpgrade
	urbit.Info.SizeLimit = inputs.dockerConfig.SizeLimit
	urbit.Info.RemoteTlonBackupsEnabled = inputs.dockerConfig.RemoteTlonBackup
	urbit.Info.LocalTlonBackupsEnabled = inputs.dockerConfig.LocalTlonBackup
	urbit.Info.DisableShipRestarts = inputs.disableShipRestarts
	urbit.Info.BackupTime = inputs.dockerConfig.BackupTime
	urbit.Info.SnapTime = inputs.dockerConfig.SnapTime

	if remoteBak, exists := backups.remote[pier]; exists {
		urbit.Info.RemoteTlonBackups = remoteBak
	}
	if localDailyBak, exists := backups.localDaily[pier]; exists {
		urbit.Info.LocalDailyTlonBackups = localDailyBak
	}
	if localWeeklyBak, exists := backups.localWeekly[pier]; exists {
		urbit.Info.LocalWeeklyTlonBackups = localWeeklyBak
	}
	if localMonthlyBak, exists := backups.localMonthly[pier]; exists {
		urbit.Info.LocalMonthlyTlonBackups = localMonthlyBak
	}

	return urbit
}

type appInfoCollector struct{}

func (appInfoCollector) collect(runtimeRt collectorRuntime) structs.Apps {
	var apps structs.Apps
	settings := runtimeRt.penpaiSettings()
	var modelTitles []string
	for _, penpaiInfo := range settings.Models {
		modelTitles = append(modelTitles, penpaiInfo.ModelTitle)
	}
	apps.Penpai.Info.Models = modelTitles
	apps.Penpai.Info.Allowed = settings.Allowed
	apps.Penpai.Info.ActiveModel = settings.ActiveModel
	apps.Penpai.Info.Running = settings.Running
	apps.Penpai.Info.MaxCores = runtime.NumCPU() - 1
	apps.Penpai.Info.ActiveCores = settings.ActiveCores
	return apps
}

type profileInfoCollector struct{}

func (profileInfoCollector) collect(runtimeRt collectorRuntime, regions map[string]structs.StartramRegion) structs.Profile {
	var startramInfo structs.Startram
	settings := runtimeRt.startramSettings()
	startramInfo.Info.Registered = settings.WgRegistered
	startramInfo.Info.Running = settings.WgOn
	startramInfo.Info.Endpoint = settings.EndpointURL
	startramInfo.Info.RemoteBackupReady = settings.RemoteBackupPassword != ""
	startramInfo.Info.BackupTime = runtimeRt.backupTime().Format("3:04PM MST")

	startramConfig := runtimeRt.startramConfig()
	startramInfo.Info.Region = startramConfig.Region
	startramInfo.Info.Expiry = startramConfig.Lease
	startramInfo.Info.Renew = startramConfig.Ongoing == 0
	startramInfo.Info.UrlID = startramConfig.UrlID

	startramServices := []string{}
	for _, subdomain := range startramConfig.Subdomains {
		parts := strings.Split(subdomain.URL, ".")
		if len(parts) < 3 {
			zap.L().Warn(fmt.Sprintf("startram services information invalid url: %s", subdomain.URL))
			continue
		}
		patp := parts[len(parts)-3]
		shipExists := false
		for _, ship := range startramServices {
			if ship == patp {
				shipExists = true
				break
			}
		}
		if !shipExists {
			startramServices = append(startramServices, patp)
		}
	}
	startramInfo.Info.StartramServices = startramServices
	startramInfo.Info.Regions = regions

	var profile structs.Profile
	profile.Startram = startramInfo
	return profile
}

type systemInfoCollector struct{}

func (systemInfoCollector) collect(runtimeRt collectorRuntime) structs.System {
	var ramObj []uint64
	var sysInfo structs.System
	sysInfo.Info.Updates = system.SystemUpdates
	sysInfo.Info.Wifi = system.WifiInfoSnapshot()
	smart := system.SmartResultsSnapshot()

	if usedRam, totalRam, err := system.GetMemory(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting memory usage: %v", err))
	} else {
		sysInfo.Info.Usage.RAM = append(ramObj, usedRam, totalRam)
	}
	if cpuUsage, err := system.GetCPU(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting CPU usage: %v", err))
	} else {
		sysInfo.Info.Usage.CPU = cpuUsage
	}
	if cpuTemp, err := system.GetTemp(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error reading CPU temperature: %v", err))
	} else {
		sysInfo.Info.Usage.CPUTemp = cpuTemp
	}
	if diskUsage, err := system.GetDisk(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting disk usage: %v", err))
	} else {
		sysInfo.Info.Usage.Disk = diskUsage
		sysInfo.Info.Usage.SwapFile = runtimeRt.swapSettings().SwapVal
	}
	drives := make(map[string]structs.SystemDrive)
	if blockDevices, err := system.ListHardDisks(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting block devices: %v", err))
	} else {
		for _, dev := range blockDevices.BlockDevices {
			if strings.HasPrefix(dev.Name, "mmcblk") {
				continue
			}
			if len(dev.Children) < 1 {
				if system.IsDevMounted(dev) {
					re := regexp.MustCompile(`^/groundseg-(\d+)$`)
					matches := re.FindStringSubmatch(dev.Mountpoints[0])
					if len(matches) > 1 {
						n, err := strconv.Atoi(matches[1])
						if err != nil {
							continue
						}
						drives[dev.Name] = structs.SystemDrive{
							DriveID: n,
						}
					}
				} else {
					drives[dev.Name] = structs.SystemDrive{
						DriveID: 0,
					}
				}
			}
		}
	}
	sysInfo.Info.SMART = smart
	sysInfo.Info.Drives = drives
	return sysInfo
}

func getContainerNetworks(containers []string) map[string]string {
	return getContainerNetworksWithLookup(defaultCollectorRuntime().getContainerNetworkFn, containers)
}

func getContainerNetworksWithLookup(getContainerNetworkFn func(string) (string, error), containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := getContainerNetworkFn(container)
		if err != nil {
			continue
		} else {
			res[container] = network
		}
	}
	return res
}
