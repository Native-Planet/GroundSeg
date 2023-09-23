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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	hostInfoInterval  = 1 * time.Second // how often we refresh system info
	shipInfoInterval  = 1 * time.Second // how often we refresh ship info
	broadcastState    structs.AuthBroadcast
	UrbitTransitions  = make(map[string]structs.UrbitTransitionBroadcast)
	SysTransBus       = make(chan structs.SystemTransitionBroadcast, 100)
	SystemTransitions structs.SystemTransitionBroadcast
	UrbTransMu        sync.RWMutex
	SysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
)

func init() {
	// initialize broadcastState global var
	if err := bootstrapBroadcastState(); err != nil {
		panic(fmt.Sprintf("Unable to initialize broadcast: %v", err))
	}
	go WsDigester()
}

func WsDigester() {
	for {
		event := <-structs.WsEventBus
		if event.Conn.Conn != nil {
			if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
				logger.Logger.Warn(fmt.Sprintf("WS error: %v", err))
				if err = auth.WsNilSession(event.Conn.Conn); err != nil {
					logger.Logger.Warn("Couldn't remove WS session")
				}
				continue
			}
		}
	}
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
		return fmt.Errorf("Couldn't get StarTram regions: %v", err)
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
	hostName, err := os.Hostname()
	if err != nil {
		errmsg := fmt.Sprintf("Error getting hostname, defaulting to `nativeplanet`: %v", err)
		logger.Logger.Warn(errmsg)
		hostName = "nativeplanet"
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
		dockerStats, err = docker.GetContainerStats(pier)
		if err != nil {
			//errmsg := fmt.Sprintf("Unable to load %s stats: %v", pier, err) // temp surpress
			//logger.Logger.Error(errmsg)
			continue
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
		urbitURL := fmt.Sprintf("http://%s.local:%d", hostName, dockerConfig.HTTPPort)
		if dockerConfig.Network == "wireguard" {
			urbitURL = fmt.Sprintf("https://%s", dockerConfig.WgURL)
			setRemote = true
		}
		minIOUrl := fmt.Sprintf("https://console.s3.%s", dockerConfig.WgURL)
		minIOPwd := ""
		if conf.WgRegistered && conf.WgOn {
			minIOPwd, err = config.GetMinIOPassword(fmt.Sprintf("minio_%s", pier))
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to get MinIO Password: %v", err))
			}
		}
		var lusCode string
		if strings.Contains(pierStatus[pier], "Up") {
			lusCode, _ = click.GetLusCode(pier)
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
	regions, err := startram.GetRegions()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't get StarTram regions: %v", err))
	} else {
		startramInfo.Info.Regions = regions
	}
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