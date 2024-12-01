package broadcast

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/leak"
	"groundseg/roller"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	broadcastInterval = 1 * time.Second // how often we refresh system info
	broadcastState    structs.AuthBroadcast
	scheduledPacks    = make(map[string]time.Time)
	UrbitTransitions  = make(map[string]structs.UrbitTransitionBroadcast)
	SchedulePackBus   = make(chan string)
	SystemTransitions structs.SystemTransitionBroadcast
	PackMu            sync.RWMutex
	UrbTransMu        sync.RWMutex
	SysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
	BackupDir         = setBackupDir()
	PointInfo         = make(map[string]structs.Point)
)

func init() {
	// initialize broadcastState global var
	if err := bootstrapBroadcastState(); err != nil {
		panic(fmt.Sprintf("Unable to initialize broadcast: %v", err))
	}
	if err := LoadStartramRegions(); err != nil {
		zap.L().Error("Couldn't load StarTram regions")
	}
	// go WsDigester()
}

// serialized single thread for ws writes (mutex instead so this isnt necessary)
// func WsDigester() {
// 	for {
// 		event := <-structs.WsEventBus
// 		if event.Conn.Conn != nil {
// 			if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
// 				zap.L().Warn(fmt.Sprintf("WS error: %v", err))
// 				if err = auth.WsNilSession(event.Conn.Conn); err != nil {
// 					zap.L().Warn("Couldn't remove WS session")
// 				}
// 				continue
// 			}
// 		}
// 	}
// }

func UpdateScheduledPack(patp string, meldNext time.Time) error {
	PackMu.Lock()
	defer PackMu.Unlock()
	scheduledPacks[patp] = meldNext
	return nil
}

func GetScheduledPack(patp string) time.Time {
	PackMu.Lock()
	defer PackMu.Unlock()
	nextPack, exists := scheduledPacks[patp]
	if !exists {
		return time.Time{}
	}
	return nextPack
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState() error {
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	// update broadcastState with ship info
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	// update with system state
	sysInfo := constructSystemInfo()
	mu.Lock()
	broadcastState.System = sysInfo
	mu.Unlock()
	// update with profile state
	profileInfo := constructProfileInfo()
	mu.Lock()
	broadcastState.Profile = profileInfo
	mu.Unlock()
	// update with apps state
	appsInfo := constructAppsInfo()
	mu.Lock()
	broadcastState.Apps = appsInfo
	mu.Unlock()
	// start looping info refreshes
	go BroadcastLoop()
	return nil
}

func GetStartramServices() error {
	zap.L().Info("Retrieving StarTram services info")
	if res, err := startram.Retrieve(); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		return err
	} else {
		zap.L().Info(fmt.Sprintf("%+v", res.Subdomains))
		return nil
	}
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	zap.L().Info("Retrieving StarTram region info")
	regions, err := startram.GetRegions()
	if err != nil {
		return err
	} else {
		mu.Lock()
		broadcastState.Profile.Startram.Info.Regions = regions
		mu.Unlock()
	}
	return nil
}

// retrieve current point info from roller
func LoadPointInfo(urbits []string) map[string]*structs.Point {
	points := make(map[string]*structs.Point)
	for _, ship := range urbits {
		point, err := roller.Client.GetPoint(config.Ctx, ship)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to get point for %s: %v", ship, err))
			continue
		}
		points[ship] = point
	}
	return points
}

// this is for building the broadcast objects describing piers
func ConstructPierInfo() (map[string]structs.Urbit, error) {
	// get a list of piers
	conf := config.Conf()
	piers := conf.Piers

	// retrieve backup information
	remoteBackups := config.StartramConfig.Backups
	remoteBackupMap := make(structs.Backup)
	for _, backup := range remoteBackups {
		for ship, backupInfo := range backup {
			remoteBackupMap[ship] = backupInfo
		}
	}
	localBackups := make(structs.Backup)
	// get local backups
	// BackupDir has been set in init
	// inside BackupDir there is a folder for each ship
	// each of those folders contains a number of numbered files with no file extension, they are all unix timestamps
	// backupInfo is a struct with a timestamp and md5
	// we want to get all the backups for a given ship and add them to localBackups
	for _, ship := range piers {
		shipBackups, err := filepath.Glob(filepath.Join(BackupDir, ship, "*"))
		if err != nil {
			continue
		}
		for _, backup := range shipBackups {
			// each backup is a path with the filename being the unix timestamp
			// strip off the dir and filename to get the timestamp
			timestamp, err := strconv.Atoi(filepath.Base(backup))
			if err != nil {
				continue
			}
			localBackups[ship] = append(localBackups[ship], structs.BackupObject{Timestamp: timestamp, MD5: ""})
		}
	}

	docker.ContainerStatList = piers
	updates := make(map[string]structs.Urbit)
	// load fresh broadcast state
	currentState := GetState()
	// get the networks containers are attached to
	shipNetworks := GetContainerNetworks(piers)
	// find out whether they're running
	pierStatus, err := docker.GetShipStatus(piers)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to bootstrap urbit states: %v", err)
		zap.L().Error(errmsg)
		return updates, err
	}
	hostName := system.LocalUrl
	if hostName == "" {
		zap.L().Debug("Defaulting to `nativeplanet.local`")
		hostName = "nativeplanet.local"
	}
	// get a sublist of piers on azimuth
	aziPiers := aziPatps(config.Conf().Piers)
	aziInfo := LoadPointInfo(aziPiers)
	// convert the running status into bools
	for pier, status := range pierStatus {
		// pull urbit info from json
		err := config.LoadUrbitConfig(pier)
		if err != nil {
			errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
			zap.L().Error(errmsg)
			continue
		}
		dockerConfig := config.UrbitConf(pier)
		// get container stats from docker

		dockerStats := docker.GetContainerStats(pier)
		urbit := structs.Urbit{}
		if existingUrbit, exists := currentState.Urbits[pier]; exists {
			// If the ship already exists in broadcastState, use its current state
			urbit = existingUrbit
		}
		isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
		bootStatus := true
		if dockerConfig.BootStatus == "ignore" {
			bootStatus = false
		}
		setRemote := false
		urbitURL := fmt.Sprintf("http://%s:%d", hostName, dockerConfig.HTTPPort)
		if dockerConfig.Network == "wireguard" {
			urbitURL = fmt.Sprintf("https://%s", dockerConfig.WgURL)
			setRemote = true
		}
		remoteReady := false
		for _, subdomain := range config.StartramConfig.Subdomains {
			if subdomain.URL == dockerConfig.WgURL {
				if subdomain.Status == "ok" {
					remoteReady = true
				}
			}
		}
		urbitAlias := dockerConfig.CustomUrbitWeb
		minIOAlias := dockerConfig.CustomS3Web
		showUrbAlias := false
		if dockerConfig.ShowUrbitWeb == "custom" {
			showUrbAlias = true
		}
		minIOUrl := fmt.Sprintf("https://console.s3.%s", dockerConfig.WgURL)
		minIOPwd := ""
		if conf.WgRegistered && conf.WgOn {
			minIOPwd, err = config.GetMinIOPassword(fmt.Sprintf("minio_%s", pier))
			if err != nil {
				//zap.L().Debug(fmt.Sprintf("Failed to get MinIO Password: %v", err))
			}
		}
		var lusCode string
		if strings.Contains(pierStatus[pier], "Up") {
			lusCode, _ = click.GetLusCode(pier)
		}

		minioLinked := config.GetMinIOLinkedStatus(pier)

		var penpaiCompanionInstalled bool
		if strings.Contains(pierStatus[pier], "Up") {
			deskStatus, err := click.GetDesk(pier, "penpai", false)
			if err != nil {
				penpaiCompanionInstalled = false
				zap.L().Debug(fmt.Sprintf("Broadcast failed to get penpai desk info for %v: %v", pier, err))
			}
			penpaiCompanionInstalled = deskStatus == "running"
		}

		var gallsegInstalled bool
		if strings.Contains(pierStatus[pier], "Up") {
			deskStatus, err := click.GetDesk(pier, "groundseg", false)
			if err != nil {
				gallsegInstalled = false
				zap.L().Debug(fmt.Sprintf("Broadcast failed to get groundseg desk info for %v: %v", pier, err))
			}
			gallsegInstalled = deskStatus == "running"
		}

		startramReminder := true
		if dockerConfig.StartramReminder == false {
			startramReminder = false
		}

		chopOnUpgrade := true
		if dockerConfig.ChopOnUpgrade == false {
			chopOnUpgrade = false
		}

		// pack day
		days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
		packDay := "Monday"
		for _, v := range days {
			if v == dockerConfig.MeldDay {
				packDay = strings.Title(dockerConfig.MeldDay)
			}
		}

		// pack date
		packDate := 1
		if dockerConfig.MeldDate > 1 {
			packDate = dockerConfig.MeldDate
		}

		// collate all the info from our sources into the struct
		urbit.Info.LusCode = lusCode
		urbit.Info.Running = isRunning
		urbit.Info.Network = shipNetworks[pier]
		urbit.Info.URL = urbitURL
		urbit.Info.LoomSize = dockerConfig.LoomSize
		urbit.Info.DiskUsage = dockerStats.DiskUsage
		urbit.Info.MemUsage = dockerStats.MemoryUsage
		urbit.Info.DevMode = dockerConfig.DevMode
		urbit.Info.Vere = dockerConfig.UrbitVersion
		urbit.Info.DetectBootStatus = bootStatus
		urbit.Info.Remote = setRemote
		urbit.Info.RemoteReady = remoteReady
		urbit.Info.Vere = dockerConfig.UrbitVersion
		urbit.Info.MinIOUrl = minIOUrl
		urbit.Info.MinIOPwd = minIOPwd
		urbit.Info.UrbitAlias = urbitAlias
		urbit.Info.MinIOAlias = minIOAlias
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
		urbit.Info.BackupTime = dockerConfig.BackupTime
		if remoteBak, exists := remoteBackupMap[pier]; exists {
			urbit.Info.RemoteTlonBackups = remoteBak
		}
		if localBak, exists := localBackups[pier]; exists {
			urbit.Info.LocalTlonBackups = localBak
		}
		// if it has azimuth info, throw it in
		_, ok := aziInfo[pier]
		if ok {
			urbit.Info.PointInfo = aziInfo[pier]
		}
		//urbit.Info.Backups = backups
		UrbTransMu.RLock()
		urbit.Transition = UrbitTransitions[pier]
		UrbTransMu.RUnlock()

		// and insert the struct into the map we will use as input for the broadcast struct
		updates[pier] = urbit
	}
	return updates, nil
}

func constructAppsInfo() structs.Apps {
	var apps structs.Apps
	conf := config.Conf()

	// penpai
	var modelTitles []string
	// Iterate through penpais to extract modelTitle
	for _, penpaiInfo := range conf.PenpaiModels {
		modelTitles = append(modelTitles, penpaiInfo.ModelTitle)
	}
	apps.Penpai.Info.Models = modelTitles
	apps.Penpai.Info.Allowed = conf.PenpaiAllow
	apps.Penpai.Info.ActiveModel = conf.PenpaiActive
	apps.Penpai.Info.Running = conf.PenpaiRunning
	apps.Penpai.Info.MaxCores = runtime.NumCPU() - 1
	apps.Penpai.Info.ActiveCores = conf.PenpaiCores
	return apps
}

func constructProfileInfo() structs.Profile {
	// Build startram struct
	var startramInfo structs.Startram
	// Information from config
	conf := config.Conf()
	startramInfo.Info.Registered = conf.WgRegistered
	startramInfo.Info.Running = conf.WgOn
	startramInfo.Info.Endpoint = conf.EndpointUrl
	startramInfo.Info.RemoteBackupReady = conf.RemoteBackupPassword != ""
	startramInfo.Info.BackupTime = config.BackupTime.Format("3:04PM MST")
	// Information from startram
	startramInfo.Info.Region = config.StartramConfig.Region
	startramInfo.Info.Expiry = config.StartramConfig.Lease
	startramInfo.Info.Renew = config.StartramConfig.Ongoing == 0
	startramInfo.Info.UrlID = config.StartramConfig.UrlID

	startramServices := []string{}
	for _, subdomain := range config.StartramConfig.Subdomains {
		// examples of subdomain.URL
		/*
			console.s3.wolryx-rosbyn-nallux-dozryl.nativeplanet.live
			console.s3.worbep-halrec-nallux-dozryl.nativeplanet.live
			s3.watmyl-ponrup-nallux-dozryl.nativeplanet.live
			s3.wolryx-rosbyn-nallux-dozryl.nativeplanet.live
			fadlyn-rivsul-nallux-dozryl.nativeplanet.live
			ames.lablet-nallux-dozryl.nativeplanet.live
			ames.ladsec-rinwyt-nallux-dozryl.nativeplanet.live
			bucket.s3.wolryx-rosbyn-nallux-dozryl.nativeplanet.live
			bucket.s3.worbep-halrec-nallux-dozryl.nativeplanet.live
			console.s3.fadlyn-rivsul-nallux-dozryl.nativeplanet.live
			console.s3.lablet-nallux-dozryl.nativeplanet.live
		*/
		//startramServices := make(map[string]structs.StartramService)
		parts := strings.Split(subdomain.URL, ".")
		// Check if there are at least three elements
		if len(parts) < 3 {
			zap.L().Warn(fmt.Sprintf("startram services information invalid url: %s", subdomain.URL))
			continue
		} else {
			// Select the third last item
			patp := parts[len(parts)-3]

			// Put ships in slice
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
	}
	startramInfo.Info.StartramServices = startramServices

	// Get Regions
	startramInfo.Info.Regions = broadcastState.Profile.Startram.Info.Regions
	// Build profile struct
	var profile structs.Profile
	profile.Startram = startramInfo
	return profile
}

// put together the system[usage] subobject
func constructSystemInfo() structs.System {
	// init
	var ramObj []uint64
	var sysInfo structs.System
	conf := config.Conf()

	// Linux update
	sysInfo.Info.Updates = system.SystemUpdates

	// Wifi
	sysInfo.Info.Wifi = system.WifiInfo
	// Sys details
	usedRam, totalRam := system.GetMemory()
	sysInfo.Info.Usage.RAM = append(ramObj, usedRam, totalRam)
	sysInfo.Info.Usage.CPU = system.GetCPU()
	sysInfo.Info.Usage.CPUTemp = system.GetTemp()
	if diskUsage, err := system.GetDisk(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting disk usage: %v", err))
	} else {
		sysInfo.Info.Usage.Disk = diskUsage
		sysInfo.Info.Usage.SwapFile = conf.SwapVal
	}
	drives := make(map[string]structs.SystemDrive)
	// Block Devices
	if blockDevices, err := system.ListHardDisks(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting block devices: %v", err))
	} else {
		for _, dev := range blockDevices.BlockDevices {
			if strings.HasPrefix(dev.Name, "mmcblk") {
				continue
			}
			// groundseg formatted drives do not have partitions
			if len(dev.Children) < 1 {
				// is the device mounted?
				if system.IsDevMounted(dev) {
					// check if mountpoint meets convention
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
				} else { // device not mounted
					drives[dev.Name] = structs.SystemDrive{
						DriveID: 0, // default value, only for empty
					}
				}
			}
			sysInfo.Info.SMART = system.SmartResults
		}
	}
	sysInfo.Info.Drives = drives

	// Transition
	sysInfo.Transition = SystemTransitions

	return sysInfo
}

// return a map of ships and their networks
func GetContainerNetworks(containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := docker.GetContainerNetwork(container)
		if err != nil {
			//errmsg := fmt.Sprintf("Error getting container network: %v", err)
			//zap.L().Error(errmsg) // temp surpress
			continue
		} else {
			res[container] = network
		}
	}
	return res
}

// stupid update method instead of psychotic recursion
func UpdateBroadcast(broadcast structs.AuthBroadcast) {
	mu.Lock()
	defer mu.Unlock()
	broadcastState = broadcast
}

// return broadcast state
func GetState() structs.AuthBroadcast {
	mu.RLock()
	defer mu.RUnlock()
	return broadcastState
}

// return json string of current broadcast state
func GetStateJson(bState structs.AuthBroadcast) ([]byte, error) {
	//temp
	bState.Type = "structure"
	bState.AuthLevel = "authorized"
	//end temp
	broadcastJson, err := json.Marshal(bState)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling response: %v", err)
		zap.L().Error(errmsg)
		return nil, err
	}
	return broadcastJson, nil
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	bState := GetState()
	leak.LeakChan <- bState // vere 3.0
	cm := auth.GetClientManager()
	if cm.HasAuthSession() {
		authJson, err := GetStateJson(bState)
		auth.ClientManager.BroadcastAuth(authJson)
		if err != nil {
			return err
		}
		auth.ClientManager.BroadcastAuth(authJson)
		return nil
	}
	return nil
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	auth.ClientManager.BroadcastUnauth(input)
	return nil
}
func ReloadUrbits() error {
	zap.L().Info("Reloading ships in broadcast")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	return nil
}

func setBackupDir() string {
	mmc, _ := isMountedMMC(config.BasePath)
	if mmc {
		return "/media/data/backup"
	} else {
		return filepath.Join(config.BasePath, "backup")
	}

}

func isMountedMMC(dirPath string) (bool, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions")
	}
	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == dirPath {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return true, nil
				} else {
					break OuterLoop
				}
			}
		}
		if dirPath == "/" {
			break
		}
		dirPath = path.Dir(dirPath) // Reduce the path by one level
	}
	return false, nil
}

// cant have more than one hyphen in @p and be on azimuth (black guy pointing at his head and smirking)
func aziPatps(ships []string) []string {
	filtered := make([]string, 0, len(ships))
	for _, str := range ships {
		count := strings.Count(str, "-")
		if count <= 1 {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
