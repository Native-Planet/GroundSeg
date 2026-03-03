package collectors

import (
	"fmt"
	"groundseg/structs"
	"groundseg/system"
	"groundseg/transition"
	"strings"
	"time"

	"go.uber.org/zap"
)

type pierRuntimeSnapshot struct {
	currentState map[string]structs.Urbit
	shipNetworks map[string]string
	pierStatus   map[string]string
	hostName     string
}

func runtimeSnapshotForPiers(piers []string, urbits map[string]structs.Urbit, runtime ...collectorRuntime) (pierRuntimeSnapshot, error) {
	return runtimeSnapshotForPiersWithRuntime(collectorRuntimeOrDefault(runtime...), piers, urbits)
}

func runtimeSnapshotForPiersWithRuntime(runtime collectorRuntime, piers []string, urbits map[string]structs.Urbit) (pierRuntimeSnapshot, error) {
	snapshot := pierRuntimeSnapshot{
		currentState: urbits,
		shipNetworks: getContainerNetworksWithLookup(runtime.GetContainerNetworkFn, piers),
		hostName:     resolveBroadcastHostName(),
	}
	pierStatus, err := runtime.GetContainerShipStatusFn(piers)
	if err != nil {
		return snapshot, fmt.Errorf("getting container ship status: %w", err)
	}
	snapshot.pierStatus = pierStatus
	return snapshot, nil
}

func resolveBroadcastHostName() string {
	hostName := system.LocalUrl()
	if hostName == "" {
		zap.L().Debug("Defaulting to `nativeplanet.local`")
		hostName = "nativeplanet.local"
	}
	return hostName
}

func collectUrbitDeploymentInputsWithRuntime(
	runtime collectorRuntime,
	pier string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) (urbitDeploymentInputs, bool) {
	if err := runtime.LoadUrbitConfigFn(pier); err != nil {
		errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
		zap.L().Error(errmsg)
		return urbitDeploymentInputs{}, false
	}

	dockerConfig := runtime.UrbitConfFn(pier)
	remote := dockerConfig.Network == "wireguard"
	url := fmt.Sprintf("http://%s:%d", hostName, dockerConfig.HTTPPort)
	if remote {
		url = fmt.Sprintf("https://%s", dockerConfig.WgURL)
	}

	minIOPwd := dockerConfig.MinioPassword
	if wireguardCtx.registered && wireguardCtx.on {
		retrievedPwd, err := runtime.GetMinIOPasswordFn(fmt.Sprintf("minio_%s", pier))
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to refresh MinIO password for %s: %v", pier, err))
		} else {
			minIOPwd = retrievedPwd
		}
	}
	packDay, packDate := normalizePackSchedule(dockerConfig.MeldDay, dockerConfig.MeldDate)
	minIOLinked := runtime.GetMinIOLinkedStatusFn(pier)

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
	runtime ...collectorRuntime,
) (urbitDeploymentInputs, bool) {
	return collectUrbitDeploymentInputsWithRuntime(
		collectorRuntimeOrDefault(runtime...),
		pier,
		hostName,
		wireguardCtx,
		startramSnapshot,
	)
}

func collectUrbitDeploymentInputsForPiers(
	piers []string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
	runtime ...collectorRuntime,
) map[string]urbitDeploymentInputs {
	return collectUrbitDeploymentInputsForPiersWithRuntime(
		collectorRuntimeOrDefault(runtime...),
		piers,
		hostName,
		wireguardCtx,
		startramSnapshot,
	)
}

func collectUrbitDeploymentInputsForPiersWithRuntime(
	runtime collectorRuntime,
	piers []string,
	hostName string,
	wireguardCtx wireguardContext,
	startramSnapshot pierStartramSnapshot,
) map[string]urbitDeploymentInputs {
	inputs := make(map[string]urbitDeploymentInputs, len(piers))
	for _, pier := range piers {
		deploymentInputs, ok := collectUrbitDeploymentInputsWithRuntime(runtime, pier, hostName, wireguardCtx, startramSnapshot)
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

func collectUrbitRuntimeInputsWithRuntime(
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
		containerImageTag: func() (string, error) { return runtime.GetContainerImageTagFn(pier) },
		dockerStats:       runtime.GetContainerStatsFn(pier),
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
	runtime ...collectorRuntime,
) urbitRuntimeInputs {
	return collectUrbitRuntimeInputsWithRuntime(
		collectorRuntimeOrDefault(runtime...),
		pier,
		status,
		rtContext,
		getScheduledPack,
	)
}

func collectUrbitRuntimeInputsForPiers(
	pierStatus map[string]string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
	runtime ...collectorRuntime,
) map[string]urbitRuntimeInputs {
	resolvedRuntime := collectorRuntimeOrDefault(runtime...)
	inputs := make(map[string]urbitRuntimeInputs, len(pierStatus))
	for pier, status := range pierStatus {
		inputs[pier] = collectUrbitRuntimeInputsWithRuntime(
			resolvedRuntime,
			pier,
			status,
			rtContext,
			getScheduledPack,
		)
	}
	return inputs
}

func collectUrbitRuntimeInputsForPiersWithRuntime(
	pierStatus map[string]string,
	rtContext urbitRuntimeContext,
	getScheduledPack func(string) time.Time,
	runtime collectorRuntime,
) map[string]urbitRuntimeInputs {
	inputs := make(map[string]urbitRuntimeInputs, len(pierStatus))
	for pier, status := range pierStatus {
		inputs[pier] = collectUrbitRuntimeInputsWithRuntime(runtime, pier, status, rtContext, getScheduledPack)
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
	lusCode, err := runtime.LusCodeFn(pier)
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
	deskStatus, err := runtime.GetDeskFn(pier, desk, false)
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

func getContainerNetworks(containers []string) map[string]string {
	return getContainerNetworksWithLookup(defaultCollectorRuntime().GetContainerNetworkFn, containers)
}

func getContainerNetworksWithLookup(GetContainerNetworkFn func(string) (string, error), containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := GetContainerNetworkFn(container)
		if err != nil {
			continue
		}
		res[container] = network
	}
	return res
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
