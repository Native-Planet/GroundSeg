package broadcast

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
	"goseg/system"
	"math"
	"os"

	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	clients           = make(map[*websocket.Conn]bool)
	hostInfoInterval  = 1 * time.Second // how often we refresh system info
	shipInfoInterval  = 1 * time.Second // how often we refresh ship info
	broadcastState    structs.AuthBroadcast
	unauthState       structs.UnauthBroadcast
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
		if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
			// logger.Logger.Warn(fmt.Sprintf("WS error: %v", err))
			continue
		}
	}
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState() error {
	logger.Logger.Info("Bootstrapping state")
	// this returns a map of ship:running status
	logger.Logger.Info("Resolving pier status")
	urbits, err := constructPierInfo()
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
	//go profileStatusLoop()
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
func constructPierInfo() (map[string]structs.Urbit, error) {
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
		if dockerConfig.Network == "wireguard" {
			setRemote = true
		}
		// collate all the info from our sources into the struct
		urbit.Info.Running = isRunning
		urbit.Info.Network = shipNetworks[pier]
		urbit.Info.URL = fmt.Sprintf("http://%s.local:%d", hostName, dockerConfig.HTTPPort)
		urbit.Info.LoomSize = int(math.Pow(2, float64(dockerConfig.LoomSize)) / math.Pow(1024, 2))
		urbit.Info.DiskUsage = dockerStats.DiskUsage
		urbit.Info.MemUsage = dockerStats.MemoryUsage
		urbit.Info.DevMode = dockerConfig.DevMode
		urbit.Info.Vere = dockerConfig.UrbitVersion
		urbit.Info.DetectBootStatus = bootStatus
		urbit.Info.Remote = setRemote
		urbit.Info.Vere = dockerConfig.UrbitVersion
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
	startramInfo.Info.Expiry = nil // temp
	startramInfo.Info.Renew = true // temp

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
	sysInfo.Info.Usage.SwapFile = system.ActiveSwap(config.BasePath)
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
			//logger.Logger.Error(errmsg) // temp surpress
			continue
		} else {
			res[container] = network
		}
	}
	return res
}

/*
// update broadcastState with a map of items
// old method that sucks
func UpdateBroadcastState(values map[string]interface{}) error {
	mu.Lock()
	v := reflect.ValueOf(&broadcastState).Elem()
	for key, value := range values {
		field := v.FieldByName(key)
		if !field.IsValid() || !field.CanSet() {
			mu.Unlock()
			return fmt.Errorf("field %s does not exist or is not settable", key)
		}
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Interface {
			val = val.Elem() // Extract the underlying value from the interface
		}
		if err := recursiveUpdate(field, val); err != nil {
			mu.Unlock()
			return fmt.Errorf("error updating field %s: %v", key, err)
			return err
		}
	}
	mu.Unlock()
	BroadcastToClients()
	return nil
}

// this allows us to insert stuff into nested structs/keys and not overwrite the existing contents
// do not use this, it can't overwrite nested structs for some reason
func recursiveUpdate(dst, src reflect.Value) error {
	if !dst.CanSet() {
		return fmt.Errorf("field (type: %s, kind: %s) is not settable", dst.Type(), dst.Kind())
	}

	// If both dst and src are structs, overwrite dst with src
	if dst.Kind() == reflect.Struct && src.Kind() == reflect.Struct {
		dst.Set(src)
		return nil
	}

	// If dst is a struct and src is a map, handle them field by field
	if dst.Kind() == reflect.Struct && src.Kind() == reflect.Map {
		for _, key := range src.MapKeys() {
			dstField := dst.FieldByName(key.String())
			if !dstField.IsValid() {
				return fmt.Errorf("field %s does not exist in the struct", key.String())
			}
			// Initialize the map if it's nil and we're trying to set a map
			if dstField.Kind() == reflect.Map && dstField.IsNil() && src.MapIndex(key).Kind() == reflect.Map {
				dstField.Set(reflect.MakeMap(dstField.Type()))
			}
			if !dstField.CanSet() {
				return fmt.Errorf("field %s is not settable in the struct", key.String())
			}
			srcVal := src.MapIndex(key)
			if srcVal.Kind() == reflect.Interface {
				srcVal = srcVal.Elem()
			}
			if err := recursiveUpdate(dstField, srcVal); err != nil {
				return err
			}
		}
		return nil
	}

	// If both dst and src are maps, handle them recursively
	if dst.Kind() == reflect.Map && src.Kind() == reflect.Map {
		for _, key := range src.MapKeys() {
			srcVal := src.MapIndex(key)
			// If the key doesn't exist in dst, initialize it
			dstVal := dst.MapIndex(key)
			if !dstVal.IsValid() {
				dstVal = reflect.New(dst.Type().Elem()).Elem()
			}
			// Recursive call to handle potential nested maps or structs
			if err := recursiveUpdate(dstVal, srcVal); err != nil {
				return err
			}
			// Initialize the map if it's nil
			if dst.IsNil() {
				dst.Set(reflect.MakeMap(dst.Type()))
			}
			dst.SetMapIndex(key, dstVal)
		}
		return nil
	}

	// For non-map or non-struct fields, or for direct updates
	if dst.Type() != src.Type() {
		return fmt.Errorf("type mismatch: expected %s, got %s", dst.Type(), src.Type())
	}
	dst.Set(src)
	return nil
}
*/

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
			updates, err := constructPierInfo()
			if err != nil {
				logger.Logger.Warn(fmt.Sprintf("Unable to build pier info: %v", err))
				continue
			}
			mu.RLock()
			newState := broadcastState
			mu.RUnlock()
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
			newState.Profile = updates
			UpdateBroadcast(newState)
			BroadcastToClients()
		}
	}
}
