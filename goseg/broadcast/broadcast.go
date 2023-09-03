package broadcast

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/config"
	"goseg/docker"
	"goseg/startram"
	"goseg/structs"
	"goseg/system"
	"math"
	"os"
	"reflect"
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
	SystemTransitions structs.SystemTransitionBroadcast
	UrbTransMu        sync.RWMutex
	SysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
)

func init() {
	// initialize broadcastState global var
	conf := config.Conf()
	broadcast, err := bootstrapBroadcastState(conf)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to initialize broadcast: %v", err)
		panic(errmsg)
	}
	broadcastState = broadcast
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState(conf structs.SysConfig) (structs.AuthBroadcast, error) {
	config.Logger.Info("Bootstrapping state")
	var res structs.AuthBroadcast
	// get a list of piers from config
	piers := conf.Piers
	// this returns a map of ship:running status
	config.Logger.Info("Resolving pier status")
	updates, err := constructPierInfo(piers)
	if err != nil {
		return res, err
	}
	// update broadcastState
	err = UpdateBroadcastState(map[string]interface{}{
		"Urbits": updates,
	})
	if err != nil {
		errmsg := fmt.Sprintf("Unable to update broadcast state: %v", err)
		config.Logger.Error(errmsg)
		return res, err
	}
	// get startram regions
	if err := LoadStartramRegions(); err != nil {
		config.Logger.Warn("%v", err)
	}
	// update with system state
	sysInfo := constructSystemInfo()
	err = UpdateBroadcastState(sysInfo)
	if err != nil {
		errmsg := fmt.Sprintf("Error updating broadcast state:", err)
		config.Logger.Error(errmsg)
	}
	// start looping info refreshes
	go hostStatusLoop()
	go shipStatusLoop()
	// return the boostrapped result
	res = GetState()
	return res, nil
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	config.Logger.Info("Retrieving StarTram region info")
	regions, err := startram.GetRegions()
	if err != nil {
		return fmt.Errorf("Couldn't get StarTram regions: %v", err)
	} else {
		mu.Lock()
		broadcastState.Profile.Startram.Info.Regions = regions
		mu.Unlock()
		BroadcastToClients()
	}
	return nil
}

// this is for building the broadcast objects describing piers
func constructPierInfo(piers []string) (map[string]structs.Urbit, error) {
	updates := make(map[string]structs.Urbit)
	// load fresh broadcast state
	currentState := GetState()
	// get the networks containers are attached to
	shipNetworks := GetContainerNetworks(piers)
	// find out whether they're running
	pierStatus, err := docker.GetShipStatus(piers)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to bootstrap urbit states: %v", err)
		config.Logger.Error(errmsg)
		return updates, err
	}
	hostName, err := os.Hostname()
	if err != nil {
		errmsg := fmt.Sprintf("Error getting hostname, defaulting to `nativeplanet`: %v", err)
		config.Logger.Warn(errmsg)
		hostName = "nativeplanet"
	}
	// convert the running status into bools
	for pier, status := range pierStatus {
		// pull urbit info from json
		err := config.LoadUrbitConfig(pier)
		if err != nil {
			errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
			config.Logger.Error(errmsg)
			continue
		}
		dockerConfig := config.UrbitConf(pier)
		// get container stats from docker
		var dockerStats structs.ContainerStats
		dockerStats, err = docker.GetContainerStats(pier)
		if err != nil {
			//errmsg := fmt.Sprintf("Unable to load %s stats: %v", pier, err) // temporary supress
			//config.Logger.Error(errmsg)
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

// put together the system[usage] subobject
func constructSystemInfo() map[string]interface{} {
	var res map[string]interface{}
	var ramObj []uint64
	var diskObj []uint64
	usedRam, totalRam := system.GetMemory()
	ramObj = append(ramObj, usedRam, totalRam)
	cpuUsage := system.GetCPU()
	cpuTemp := system.GetTemp()
	usedDisk, freeDisk := system.GetDisk()
	diskObj = append(diskObj, usedDisk, freeDisk)
	swapVal := system.HasSwap()
	res = map[string]interface{}{
		"System": map[string]interface{}{
			"Info": map[string]interface{}{
				"Usage": map[string]interface{}{
					"RAM":      ramObj,
					"CPU":      cpuUsage,
					"CPUTemp":  cpuTemp,
					"Disk":     diskObj,
					"SwapFile": swapVal,
				},
			},
			"Transition": SystemTransitions,
		},
	}
	return res
}

// return a map of ships and their networks
func GetContainerNetworks(containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := docker.GetContainerNetwork(container)
		if err != nil {
			//errmsg := fmt.Sprintf("Error getting container network: %v", err) // temporary supress
			//config.Logger.Error(errmsg)
			continue
		} else {
			res[container] = network
		}
	}
	return res
}

// update broadcastState with a map of items
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

// return broadcast state
func GetState() structs.AuthBroadcast {
	mu.Lock()
	defer mu.Unlock()
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
		config.Logger.Error(errmsg)
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
			err := UpdateBroadcastState(update)
			if err != nil {
				config.Logger.Warn(fmt.Sprintf("Error updating system status: %v", err))
			}
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
			conf := config.Conf()
			piers := conf.Piers
			updates, err := constructPierInfo(piers)
			if err != nil {
				errmsg := fmt.Sprintf("Unable to build pier info: %v", err)
				config.Logger.Warn(errmsg)
				continue
			}
			mu.Lock() // Locking the mutex
			for key, urbit := range updates {
				//broadcastState.Urbits[key] = urbit
				config.Logger.Warn(fmt.Sprintf("%+v %+v", key, urbit))
				config.Logger.Warn(fmt.Sprintf("%+v", broadcastState.Urbits))
			}
			mu.Unlock() // Unlocking the mutex
			BroadcastToClients()
		}
	}
}
