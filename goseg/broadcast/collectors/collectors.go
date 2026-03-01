package collectors

import (
	"fmt"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
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

// Dependency seams for easier testing and future runtime overrides.
func resolveBackupRoot(basePath string) string {
	return backupsvc.ResolveBackupRoot(basePath)
}

func currentBasePath() string {
	return config.BasePath
}

func startramSettingsSnapshot() config.StartramSettings {
	return config.StartramSettingsSnapshot()
}

func startramConfigSnapshot() structs.StartramRetrieve {
	return config.GetStartramConfig()
}

func loadUrbitConfig(pier string) error {
	return config.LoadUrbitConfig(pier)
}

func urbitConfig(pier string) structs.UrbitDocker {
	return config.UrbitConf(pier)
}

func getMinIOPassword(name string) (string, error) {
	return config.GetMinIOPassword(name)
}

func isMinIOLinked(patp string) bool {
	return config.GetMinIOLinkedStatus(patp)
}

func getShipStatus(piers []string) (map[string]string, error) {
	return docker.GetShipStatus(piers)
}

func getContainerNetwork(container string) (string, error) {
	return docker.GetContainerNetwork(container)
}

func getContainerStats(container string) structs.ContainerStats {
	return docker.GetContainerStats(container)
}

func getContainerImageTag(container string) (string, error) {
	return docker.GetContainerImageTag(container)
}

func getLusCode(pier string) (string, error) {
	return click.GetLusCode(pier)
}

func getDesk(pier, desk string, bypass bool) (string, error) {
	return click.GetDesk(pier, desk, bypass)
}

func broadcastHostName() string {
	return system.LocalUrl
}

func listHardDisks() (structs.LSBLKDevice, error) {
	return system.ListHardDisks()
}

func isDevMounted(dev structs.BlockDev) bool {
	return system.IsDevMounted(dev)
}

func runtimeSnapshotCollectorImpl() runtimeSnapshotCollector {
	return runtimeSnapshotCollector{}
}

func startramRegionsFn() (map[string]structs.StartramRegion, error) {
	return startram.SyncRegions()
}

func startramRetrieveFn() (structs.StartramRetrieve, error) {
	return startram.Retrieve()
}

// ConstructPierInfo builds the urbit entries for broadcast state.
func ConstructPierInfo(currentState structs.AuthBroadcast, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	settings := startramSettingsSnapshot()
	piers := settings.Piers
	updates := make(map[string]structs.Urbit)

	backups := backupSnapshotForPiers(piers, startramConfigSnapshot().Backups)
	rtSnapshot, err := runtimeSnapshotForPiers(piers, currentState)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to bootstrap urbit states: %v", err)
		zap.L().Error(errmsg)
		return updates, err
	}
	startramSnapshot := startramSnapshotForPiers(startramConfigSnapshot().Subdomains)
	urbitInputs := map[string]urbitViewInputs{}
	for pier, status := range rtSnapshot.pierStatus {
		inputs, ok := collectUrbitViewInputs(pier, status, settings, rtSnapshot, startramSnapshot, scheduled)
		if !ok {
			continue
		}
		urbitInputs[pier] = inputs
	}

	for pier := range rtSnapshot.pierStatus {
		inputs, ok := urbitInputs[pier]
		if !ok {
			continue
		}
		urbit, ok := urbitViewMapper{}.assemble(pier, inputs, backups)
		if ok {
			updates[pier] = urbit
		}
	}
	return updates, nil
}

func ConstructAppsInfo() structs.Apps {
	return appInfoCollector{}.collect()
}

func ConstructProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	return profileInfoCollector{}.collect(regions)
}

// ConstructSystemInfo builds system usage and update health info.
func ConstructSystemInfo() structs.System {
	return systemInfoCollector{}.collect()
}

func LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	regions, err := startramRegionsFn()
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func GetStartramServices() error {
	zap.L().Info("Retrieving StarTram services info")
	res, err := startramRetrieveFn()
	if err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		return err
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
	currentState structs.AuthBroadcast
	shipNetworks map[string]string
	pierStatus   map[string]string
	hostName     string
}

type pierStartramSnapshot struct {
	remoteReadyByURL map[string]bool
}

func backupSnapshotForPiers(piers []string, remoteBackups []structs.Backup) pierBackupSnapshot {
	return pierBackupSnapshot{
		remote:       flattenRemoteBackups(remoteBackups),
		localDaily:   localBackupsForPeriod(piers, "daily"),
		localWeekly:  localBackupsForPeriod(piers, "weekly"),
		localMonthly: localBackupsForPeriod(piers, "monthly"),
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
	localBackups := make(structs.Backup)
	for _, ship := range piers {
		backupDir := resolveBackupRoot(currentBasePath())
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

func runtimeSnapshotForPiers(piers []string, currentState structs.AuthBroadcast) (pierRuntimeSnapshot, error) {
	return runtimeSnapshotCollectorImpl().collect(piers, currentState)
}

func resolveBroadcastHostName() string {
	hostName := broadcastHostName()
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

type runtimeSnapshotCollector struct{}

func (runtimeSnapshotCollector) collect(piers []string, currentState structs.AuthBroadcast) (pierRuntimeSnapshot, error) {
	snapshot := pierRuntimeSnapshot{
		currentState: currentState,
		shipNetworks: getContainerNetworks(piers),
		hostName:     resolveBroadcastHostName(),
	}
	pierStatus, err := getShipStatus(piers)
	if err != nil {
		return snapshot, err
	}
	snapshot.pierStatus = pierStatus
	return snapshot, nil
}

func collectUrbitViewInputs(
	pier string,
	status string,
	settings config.StartramSettings,
	rtSnapshot pierRuntimeSnapshot,
	startramSnapshot pierStartramSnapshot,
	getScheduledPack func(string) time.Time,
) (urbitViewInputs, bool) {
	if err := loadUrbitConfig(pier); err != nil {
		errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
		zap.L().Error(errmsg)
		return urbitViewInputs{}, false
	}

	dockerConfig := urbitConfig(pier)
	dockerStats := getContainerStats(pier)

	existing := structs.Urbit{}
	if existingUrbit, exists := rtSnapshot.currentState.Urbits[pier]; exists {
		existing = existingUrbit
	}

	isRunning := status == "Up" || strings.HasPrefix(status, "Up ")
	bootStatus := dockerConfig.BootStatus != "ignore"
	remote := dockerConfig.Network == "wireguard"
	urbitURL := fmt.Sprintf("http://%s:%d", rtSnapshot.hostName, dockerConfig.HTTPPort)
	if remote {
		urbitURL = fmt.Sprintf("https://%s", dockerConfig.WgURL)
	}

	minIOURL := fmt.Sprintf("https://console.s3.%s", dockerConfig.WgURL)
	minIOPwd := existing.Info.MinIOPwd
	if settings.WgRegistered && settings.WgOn {
		retrievedPwd, err := getMinIOPassword(fmt.Sprintf("minio_%s", pier))
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to refresh MinIO password for %s: %v", pier, err))
		} else {
			minIOPwd = retrievedPwd
		}
	}

	disableShipRestarts := !dockerConfig.DisableShipRestarts
	packDay, packDate := normalizePackSchedule(dockerConfig.MeldDay, dockerConfig.MeldDate)
	lusCode, err := lusCodeIfRunning(pier, status)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to resolve +code for %s: %v", pier, err))
	}

	packUnixTime := int64(0)
	if getScheduledPack != nil {
		packUnixTime = getScheduledPack(pier).Unix()
	}

	return urbitViewInputs{
		existingUrbit:       existing,
		dockerConfig:        dockerConfig,
		dockerStats:         dockerStats,
		network:             rtSnapshot.shipNetworks[pier],
		isRunning:           isRunning,
		bootStatus:          bootStatus,
		remote:              remote,
		url:                 urbitURL,
		remoteReady:         startramSnapshot.remoteReadyByURL[dockerConfig.WgURL],
		showUrbitWebAlias:   dockerConfig.ShowUrbitWeb == "custom",
		minIOPwd:            minIOPwd,
		disableShipRestarts: disableShipRestarts,
		lusCode:             lusCode,
		minIOLinked:         isMinIOLinked(pier),
		penpaiCompanion:     deskInstalledIfRunning(pier, status, "penpai"),
		gallsegInstalled:    deskInstalledIfRunning(pier, status, "groundseg"),
		startramReminder:    dockerConfig.StartramReminder,
		chopOnUpgrade:       dockerConfig.ChopOnUpgrade,
		packDay:             packDay,
		packDate:            packDate,
		minIOURL:            minIOURL,
		packUnixTime:        packUnixTime,
	}, true
}

func lusCodeIfRunning(pier string, status string) (string, error) {
	if !transition.IsContainerUpStatus(status) {
		return "", nil
	}
	lusCode, err := getLusCode(pier)
	if err != nil {
		return "", fmt.Errorf("unable to resolve +code for %s: %w", pier, err)
	}
	return lusCode, nil
}

func deskInstalledIfRunning(pier string, status string, desk string) bool {
	if !transition.IsContainerUpStatus(status) {
		return false
	}
	deskStatus, err := getDesk(pier, desk, false)
	if err != nil {
		zap.L().Debug(fmt.Sprintf("Broadcast failed to get %s desk info for %v: %v", desk, pier, err))
		return false
	}
	return deskStatus == string(transition.ContainerStatusRunning)
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
	if runningTag, err := getContainerImageTag(pier); err == nil {
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

func (appInfoCollector) collect() structs.Apps {
	var apps structs.Apps
	settings := config.PenpaiSettingsSnapshot()
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

func (profileInfoCollector) collect(regions map[string]structs.StartramRegion) structs.Profile {
	var startramInfo structs.Startram
	settings := config.StartramSettingsSnapshot()
	startramInfo.Info.Registered = settings.WgRegistered
	startramInfo.Info.Running = settings.WgOn
	startramInfo.Info.Endpoint = settings.EndpointURL
	startramInfo.Info.RemoteBackupReady = settings.RemoteBackupPassword != ""
	startramInfo.Info.BackupTime = config.BackupTime.Format("3:04PM MST")

	startramConfig := config.GetStartramConfig()
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

func (systemInfoCollector) collect() structs.System {
	var ramObj []uint64
	var sysInfo structs.System
	sysInfo.Info.Updates = system.SystemUpdates
	sysInfo.Info.Wifi = system.WifiInfoSnapshot()

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
		sysInfo.Info.Usage.SwapFile = config.SwapSettingsSnapshot().SwapVal
	}
	drives := make(map[string]structs.SystemDrive)
	if blockDevices, err := listHardDisks(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting block devices: %v", err))
	} else {
		for _, dev := range blockDevices.BlockDevices {
			if strings.HasPrefix(dev.Name, "mmcblk") {
				continue
			}
			if len(dev.Children) < 1 {
				if isDevMounted(dev) {
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
			sysInfo.Info.SMART = system.SmartResults
		}
	}
	sysInfo.Info.Drives = drives
	return sysInfo
}

func getContainerNetworks(containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := getContainerNetwork(container)
		if err != nil {
			continue
		} else {
			res[container] = network
		}
	}
	return res
}
