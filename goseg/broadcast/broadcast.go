package broadcast

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/click"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
	"goseg/system"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	hostInfoInterval  = 1 * time.Second // how often we refresh system info
	shipInfoInterval  = 1 * time.Second // how often we refresh ship info
	broadcastState    structs.AuthBroadcast
	scheduledPacks    = make(map[string]time.Time)
	UrbitTransitions  = make(map[string]structs.UrbitTransitionBroadcast)
	SysTransBus       = make(chan structs.SystemTransitionBroadcast, 100)
	SchedulePackBus   = make(chan string)
	SystemTransitions structs.SystemTransitionBroadcast
	PackMu            sync.RWMutex
	UrbTransMu        sync.RWMutex
	SysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
)

func init() {
	// initialize broadcastState global var
	if err := bootstrapBroadcastState(); err != nil {
		panic(fmt.Sprintf("Unable to initialize broadcast: %v", err))
	}
	if err := LoadStartramRegions(); err != nil {
		logger.Logger.Error("Couldn't load StarTram regions")
	}
	// go WsDigester()
}

// serialized single thread for ws writes (mutex instead so this isnt necessary)
// func WsDigester() {
// 	for {
// 		event := <-structs.WsEventBus
// 		if event.Conn.Conn != nil {
// 			if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
// 				logger.Logger.Warn(fmt.Sprintf("WS error: %v", err))
// 				if err = auth.WsNilSession(event.Conn.Conn); err != nil {
// 					logger.Logger.Warn("Couldn't remove WS session")
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
	logger.Logger.Info("Bootstrapping state")
	// this returns a map of ship:running status
	logger.Logger.Info("Resolving pier status")
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
	// start looping info refreshes
	go hostStatusLoop()
	go shipStatusLoop()
	go profileStatusLoop()
	return nil
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	logger.Logger.Info("Retrieving StarTram region info")
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

// this is for building the broadcast objects describing piers
func ConstructPierInfo() (map[string]structs.Urbit, error) {
	// get a list of piers
	conf := config.Conf()
	piers := conf.Piers
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
		logger.Logger.Error(errmsg)
		return updates, err
	}
	hostName := system.LocalUrl
	if hostName == "" {
		logger.Logger.Debug(fmt.Sprintf("Defaulting to `nativeplanet.local`"))
		hostName = "nativeplanet.local"
	}
	// convert the running status into bools
	for pier, status := range pierStatus {
		// pull urbit info from json
		err := config.LoadUrbitConfig(pier)
		if err != nil {
			errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
			logger.Logger.Error(errmsg)
			continue
		}
		dockerConfig := config.UrbitConf(pier)
		// get container stats from docker
		var dockerStats structs.ContainerStats
		res, ok := docker.ContainerStats[pier]
		if ok {
			dockerStats = res
		}
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
				logger.Logger.Debug(fmt.Sprintf("Failed to get MinIO Password: %v", err))
			}
		}
		var lusCode string
		if strings.Contains(pierStatus[pier], "Up") {
			lusCode, _ = click.GetLusCode(pier)
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
		urbit.Info.Vere = dockerConfig.UrbitVersion
		urbit.Info.MinIOUrl = minIOUrl
		urbit.Info.MinIOPwd = minIOPwd
		urbit.Info.UrbitAlias = urbitAlias
		urbit.Info.MinIOAlias = minIOAlias
		urbit.Info.ShowUrbAlias = showUrbAlias
		urbit.Info.PackScheduleActive = dockerConfig.MeldSchedule
		urbit.Info.PackDay = packDay
		urbit.Info.PackDate = packDate
		urbit.Info.PackTime = dockerConfig.MeldTime
		urbit.Info.LastPack = dockerConfig.MeldLast
		urbit.Info.NextPack = strconv.FormatInt(GetScheduledPack(pier).Unix(), 10)
		urbit.Info.PackIntervalType = dockerConfig.MeldScheduleType
		urbit.Info.PackIntervalValue = dockerConfig.MeldFrequency
		UrbTransMu.RLock()
		urbit.Transition = UrbitTransitions[pier]
		UrbTransMu.RUnlock()

		// and insert the struct into the map we will use as input for the broadcast struct
		updates[pier] = urbit
	}
	return updates, nil
}

func constructProfileInfo() structs.Profile {
	// Build startram struct
	var startramInfo structs.Startram
	// Information from config
	conf := config.Conf()
	startramInfo.Info.Registered = conf.WgRegistered
	startramInfo.Info.Running = conf.WgOn
	startramInfo.Info.Endpoint = conf.EndpointUrl

	// Information from startram
	startramInfo.Info.Region = config.StartramConfig.Region
	startramInfo.Info.Expiry = config.StartramConfig.Lease
	startramInfo.Info.Renew = config.StartramConfig.Ongoing == 0

	// Get Regions
	startramInfo.Info.Regions = broadcastState.Profile.Startram.Info.Regions
	// Build profile struct
	var profile structs.Profile
	profile.Startram = startramInfo
	return profile
}

// put together the system[usage] subobject
func constructSystemInfo() structs.System {
	var ramObj []uint64
	var diskObj []uint64
	var sysInfo structs.System
	usedRam, totalRam := system.GetMemory()
	sysInfo.Info.Usage.RAM = append(ramObj, usedRam, totalRam)
	sysInfo.Info.Usage.CPU = system.GetCPU()
	sysInfo.Info.Usage.CPUTemp = system.GetTemp()
	usedDisk, freeDisk := system.GetDisk()
	sysInfo.Info.Usage.Disk = append(diskObj, usedDisk, freeDisk)
	conf := config.Conf()
	sysInfo.Info.Usage.SwapFile = conf.SwapVal
	sysInfo.Info.Updates = system.SystemUpdates
	sysInfo.Transition = SystemTransitions
	sysInfo.Info.Wifi = system.WifiInfo
	return sysInfo
}

// return a map of ships and their networks
func GetContainerNetworks(containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := docker.GetContainerNetwork(container)
		if err != nil {
			//errmsg := fmt.Sprintf("Error getting container network: %v", err)
			//logger.Logger.Error(errmsg) // temp surpress
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
func GetStateJson() ([]byte, error) {
	bState := GetState()
	//temp
	bState.Type = "structure"
	bState.AuthLevel = "authorized"
	//end temp
	broadcastJson, err := json.Marshal(bState)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling response: %v", err)
		logger.Logger.Error(errmsg)
		return nil, err
	}
	return broadcastJson, nil
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	authJson, err := GetStateJson()
	if err != nil {
		return err
	}
	auth.ClientManager.BroadcastAuth(authJson)
	return nil
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	auth.ClientManager.BroadcastUnauth(input)
	return nil
}

// refresh loop for host info
func hostStatusLoop() {
	ticker := time.NewTicker(hostInfoInterval)
	for {
		select {
		case <-ticker.C:
			update := constructSystemInfo()
			mu.RLock()
			newState := broadcastState
			mu.RUnlock()
			update = PreserveSystemTransitions(newState, update)
			newState.System = update
			UpdateBroadcast(newState)
			BroadcastToClients()
		}
	}
}

// refresh loop for ship info
// for reasons beyond my iq we can't do this through the normal update func
func shipStatusLoop() {
	ticker := time.NewTicker(hostInfoInterval)
	for {
		select {
		case <-ticker.C:
			updates, err := ConstructPierInfo()
			if err != nil {
				logger.Logger.Warn(fmt.Sprintf("Unable to build pier info: %v", err))
				continue
			}
			mu.RLock()
			newState := broadcastState
			mu.RUnlock()
			updates = PreserveUrbitsTransitions(newState, updates)
			newState.Urbits = updates
			UpdateBroadcast(newState)
			BroadcastToClients()
		}
	}
}

func profileStatusLoop() {
	ticker := time.NewTicker(hostInfoInterval)
	for {
		select {
		case <-ticker.C:
			updates := constructProfileInfo()
			mu.RLock()
			newState := broadcastState
			mu.RUnlock()
			updates = PreserveProfileTransitions(newState, updates)
			newState.Profile = updates
			UpdateBroadcast(newState)
			BroadcastToClients()
		}
	}
}

func PreserveProfileTransitions(oldState structs.AuthBroadcast, newProfile structs.Profile) structs.Profile {
	newProfile.Startram.Transition = oldState.Profile.Startram.Transition
	return newProfile
}

func PreserveSystemTransitions(oldState structs.AuthBroadcast, newSystem structs.System) structs.System {
	newSystem.Transition = oldState.System.Transition
	return newSystem
}

func PreserveUrbitsTransitions(oldState structs.AuthBroadcast, newUrbits map[string]structs.Urbit) map[string]structs.Urbit {
	for k, v := range oldState.Urbits {
		urbitStruct, exists := newUrbits[k]
		if !exists {
			urbitStruct = structs.Urbit{}
		}
		urbitStruct.Transition = v.Transition
		newUrbits[k] = urbitStruct
	}
	return newUrbits
}

func ReloadUrbits() error {
	logger.Logger.Info("Reloading ships in broadcast")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	return nil
}
