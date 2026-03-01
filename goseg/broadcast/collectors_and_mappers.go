package broadcast

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"groundseg/system"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type runtimeSnapshotCollector struct{}

func (runtimeSnapshotCollector) collect(piers []string) (pierRuntimeSnapshot, error) {
	snapshot := pierRuntimeSnapshot{
		currentState: GetState(),
		shipNetworks: GetContainerNetworks(piers),
		hostName:     resolveBroadcastHostName(),
	}
	pierStatus, err := docker.GetShipStatus(piers)
	if err != nil {
		return snapshot, err
	}
	snapshot.pierStatus = pierStatus
	return snapshot, nil
}

type urbitViewMapper struct{}

func (urbitViewMapper) assemble(
	pier string,
	status string,
	settings config.StartramSettings,
	runtimeSnapshot pierRuntimeSnapshot,
	backups pierBackupSnapshot,
	startramSnapshot pierStartramSnapshot,
) (structs.Urbit, bool) {
	err := config.LoadUrbitConfig(pier)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
		zap.L().Error(errmsg)
		return structs.Urbit{}, false
	}

	dockerConfig := config.UrbitConf(pier)
	dockerStats := docker.GetContainerStats(pier)
	urbit := structs.Urbit{}
	if existingUrbit, exists := runtimeSnapshot.currentState.Urbits[pier]; exists {
		urbit = existingUrbit
	}

	isRunning := status == "Up" || strings.HasPrefix(status, "Up ")
	bootStatus := dockerConfig.BootStatus != "ignore"
	remote := dockerConfig.Network == "wireguard"
	urbitURL := fmt.Sprintf("http://%s:%d", runtimeSnapshot.hostName, dockerConfig.HTTPPort)
	if remote {
		urbitURL = fmt.Sprintf("https://%s", dockerConfig.WgURL)
	}
	remoteReady := startramSnapshot.remoteReadyByURL[dockerConfig.WgURL]

	showUrbAlias := dockerConfig.ShowUrbitWeb == "custom"
	minIOURL := fmt.Sprintf("https://console.s3.%s", dockerConfig.WgURL)
	minIOPwd := urbit.Info.MinIOPwd
	if settings.WgRegistered && settings.WgOn {
		retrievedPwd, err := config.GetMinIOPassword(fmt.Sprintf("minio_%s", pier))
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to refresh MinIO password for %s: %v", pier, err))
		} else {
			minIOPwd = retrievedPwd
		}
	}

	var disableShipRestarts bool
	if boolValue, ok := dockerConfig.DisableShipRestarts.(bool); ok {
		disableShipRestarts = !boolValue
	}

	lusCode := lusCodeIfRunning(pier, status)
	minioLinked := config.GetMinIOLinkedStatus(pier)
	penpaiCompanionInstalled := deskInstalledIfRunning(pier, status, "penpai")
	gallsegInstalled := deskInstalledIfRunning(pier, status, "groundseg")
	startramReminder := boolSettingWithDefaultTrue(dockerConfig.StartramReminder)
	chopOnUpgrade := boolSettingWithDefaultTrue(dockerConfig.ChopOnUpgrade)
	packDay, packDate := normalizePackSchedule(dockerConfig.MeldDay, dockerConfig.MeldDate)

	urbit.Info.LusCode = lusCode
	urbit.Info.Running = isRunning
	urbit.Info.Network = runtimeSnapshot.shipNetworks[pier]
	urbit.Info.URL = urbitURL
	urbit.Info.LoomSize = dockerConfig.LoomSize
	urbit.Info.DiskUsage = dockerStats.DiskUsage
	urbit.Info.MemUsage = dockerStats.MemoryUsage
	urbit.Info.DevMode = dockerConfig.DevMode
	urbit.Info.Vere = dockerConfig.UrbitVersion
	urbit.Info.DetectBootStatus = bootStatus
	urbit.Info.Remote = remote
	urbit.Info.RemoteReady = remoteReady
	urbit.Info.Vere = dockerConfig.UrbitVersion
	urbit.Info.MinIOUrl = minIOURL
	urbit.Info.MinIOPwd = minIOPwd
	urbit.Info.UrbitAlias = dockerConfig.CustomUrbitWeb
	urbit.Info.MinIOAlias = dockerConfig.CustomS3Web
	urbit.Info.ShowUrbAlias = showUrbAlias
	urbit.Info.MinIOLinked = minioLinked
	urbit.Info.PackScheduleActive = dockerConfig.MeldSchedule
	urbit.Info.PackDay = packDay
	urbit.Info.PackDate = packDate
	urbit.Info.PackTime = dockerConfig.MeldTime
	urbit.Info.LastPack = dockerConfig.MeldLast
	urbit.Info.NextPack = strconv.FormatInt(GetScheduledPack(pier).Unix(), 10)
	urbit.Info.PackIntervalType = dockerConfig.MeldScheduleType
	urbit.Info.PackIntervalValue = dockerConfig.MeldFrequency
	urbit.Info.PenpaiCompanion = penpaiCompanionInstalled
	urbit.Info.Gallseg = gallsegInstalled
	urbit.Info.StartramReminder = startramReminder
	urbit.Info.ChopOnUpgrade = chopOnUpgrade
	urbit.Info.SizeLimit = dockerConfig.SizeLimit
	urbit.Info.RemoteTlonBackupsEnabled = dockerConfig.RemoteTlonBackup
	urbit.Info.LocalTlonBackupsEnabled = dockerConfig.LocalTlonBackup
	urbit.Info.DisableShipRestarts = disableShipRestarts
	urbit.Info.BackupTime = dockerConfig.BackupTime
	urbit.Info.SnapTime = dockerConfig.SnapTime

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

	urbTransMu.RLock()
	urbit.Transition = urbitTransitions[pier]
	urbTransMu.RUnlock()
	return urbit, true
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

func (profileInfoCollector) collect() structs.Profile {
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
	startramInfo.Info.Regions = broadcastState.Profile.Startram.Info.Regions

	var profile structs.Profile
	profile.Startram = startramInfo
	return profile
}

type systemInfoCollector struct{}

func (systemInfoCollector) collect() structs.System {
	var ramObj []uint64
	var sysInfo structs.System
	swapSettings := config.SwapSettingsSnapshot()

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
	sysInfo.Info.Usage.CPUTemp = system.GetTemp()
	if diskUsage, err := system.GetDisk(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting disk usage: %v", err))
	} else {
		sysInfo.Info.Usage.Disk = diskUsage
		sysInfo.Info.Usage.SwapFile = swapSettings.SwapVal
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
			sysInfo.Info.SMART = system.SmartResults
		}
	}
	sysInfo.Info.Drives = drives
	sysInfo.Transition = systemTransitions
	return sysInfo
}
