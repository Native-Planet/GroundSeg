package config

// code for managing groundseg and container configurations

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"groundseg/system"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	// global settings config (accessed via funcs)
	globalConfig structs.SysConfig
	// base path for installation (override default with env var)
	BasePath = getBasePath()
	// only amd64 or arm64
	Architecture = getArchitecture()
	// struct of /retrieve blob
	StartramConfig structs.StartramRetrieve
	// struct of minio passwords
	minIOPasswords = make(map[string]string)
	// set with `./groundseg dev` (enables verbose logging)
	DebugMode  = false
	Ready      = false
	BackupTime time.Time
	// representation of desired/actual container states
	GSContainers = make(map[string]structs.ContainerState)
	// channel for log stream requests
	DockerDir     = defaults.DockerData("volumes") + "/"
	confPath      = filepath.Join(BasePath, "settings", "system.json")
	keyPath       = filepath.Join(BasePath, "settings", "session.key")
	isEMMCMachine bool
	confMutex     sync.Mutex
	contMutex     sync.Mutex
	versMutex     sync.Mutex
	minioPwdMutex sync.Mutex
)

// try initializing from system.json on disk
func init() {
	setDebugMode()
	BasePath = getBasePath()
	configureFixerScript()
	isEMMCMachine = checkIsEMMCMachine()
	zap.L().Info(fmt.Sprintf("Loading configs from %s", BasePath))
	confPath = filepath.Join(BasePath, "settings", "system.json")
	keyPath = filepath.Join(BasePath, "settings", "session.key")
	file, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = createDefaultConf()
		if err != nil {
			// panic if we can't create it
			zap.L().Error(fmt.Sprintf("Unable to create config! %v", err))
			fmt.Printf("Failed to create log directory: %v\n", err)
			fmt.Print("\n\n.・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n")
			fmt.Print("Please run GroundSeg as root!  \n /) /)\n( . . )\n(  >< )\n Love, Native Planet\n")
			fmt.Print(".・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n\n")
			panic("")
		}
		file, _ = os.Open(confPath)
		salt := RandString(32)
		wgPriv, wgPub, err := WgKeyGen()
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		} else {
			if err = UpdateConf(map[string]interface{}{
				"pubkey":  wgPub,
				"privkey": wgPriv,
				"salt":    salt,
				"keyfile": keyPath,
			}); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}
	}
	defer file.Close()
	_, err = os.Open(keyPath)
	if err != nil {
		// generate and insert aes & wireguard keys
		keyfile, err := os.Stat(keyPath)
		if err != nil || keyfile.Size() == 0 {
			keyContent := RandString(32)
			if err := ioutil.WriteFile(keyPath, []byte(keyContent), 0644); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't write keyfile! %v", err))
			}
		}
	}
	// read the sysconfig to memory
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&globalConfig); err != nil {
		zap.L().Error(fmt.Sprintf("Error decoding JSON: %v", err))
	}
	// add mising fields
	globalConfig = mergeConfigs(defaults.SysConfig(BasePath), globalConfig)
	// wipe the sessions on each startup
	//globalConfig.Sessions.Authorized = make(map[string]structs.SessionInfo)
	globalConfig.Sessions.Unauthorized = make(map[string]structs.SessionInfo)

	// get hash of groundseg binary
	hash, err := GetSHA256(filepath.Join(BasePath, "groundseg"))
	if err != nil {
		errmsg := fmt.Sprintf("Error getting binary sha256 hash: %v", err)
		zap.L().Error(errmsg)
	}
	globalConfig.BinHash = hash
	zap.L().Info(fmt.Sprintf("Binary sha256 hash: %v", hash))

	configMap := make(map[string]interface{})
	configBytes, err := json.Marshal(globalConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshaling JSON: %v", err)
		zap.L().Error(errmsg)
	}
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshaling JSON: %v", err)
		zap.L().Error(errmsg)
	}
	err = persistConf(configMap)
	if err != nil {
		errmsg := fmt.Sprintf("Error persisting JSON: %v", err)
		zap.L().Error(errmsg)
	}
	file, err = os.Open(confPath)
	if err != nil {
		errmsg := fmt.Sprintf("Error opening JSON: %v", err)
		zap.L().Error(errmsg)
	}
	decoder = json.NewDecoder(file)
	err = decoder.Decode(&globalConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Error decoding JSON: %v", err)
		zap.L().Error(errmsg)
	}
	// create a keyfile if you dont have one (gs1)
	conf := Conf()
	if conf.KeyFile == "" {
		keyPath := filepath.Join(BasePath, "settings", "session.key")
		keyfile, err := os.Stat(keyPath)
		if err != nil || keyfile.Size() == 0 {
			keyContent := RandString(32)
			if err := ioutil.WriteFile(keyPath, []byte(keyContent), 0644); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't write keyfile! %v", err))
			}
		}
		file, _ = os.Open(confPath)
		if err = UpdateConf(map[string]interface{}{
			"keyfile": keyPath,
		}); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	}
	go ConfChannel()
}

// return the global conf var
func Conf() structs.SysConfig {
	confMutex.Lock()
	defer confMutex.Unlock()
	return globalConfig
}

func checkIsEMMCMachine() bool {
	partitions, err := disk.Partitions(true)
	if err != nil {
		fmt.Println("Failed to get partitions, defaulting to non-eMMC assumption")
		return false
	}

	// Check if root partition is on eMMC
	for _, p := range partitions {
		if p.Mountpoint == "/" {
			return strings.Contains(p.Device, "mmc")
		}
	}

	// If we can't find the root partition, check if /media/data exists as a fallback
	if _, err := os.Stat("/media/data"); err == nil {
		return true
	}

	return false
}

func GetStoragePath(operation string) string {
	basePath := os.Getenv("GS_BASE_PATH")
	if basePath == "" {
		basePath = "/opt/nativeplanet/groundseg"
	}
	if !strings.HasPrefix(basePath, "/") {
		fmt.Println("Base path is not absolute! Using default")
		basePath = "/opt/nativeplanet/groundseg"
	}
	var operationPaths = map[string]string{
		"uploads": "uploads",
		"temp":    "temp",
		"exports": "exports",
		"logs":    "logs",
	}
	opPath, exists := operationPaths[operation]
	if !exists {
		fmt.Printf("Invalid operation '%s' for GetStoragePath\n", operation)
		return filepath.Join(basePath, "temp") // Default to temp
	}
	var storagePath string
	if isEMMCMachine {
		storagePath = filepath.Join("/media/data", opPath)
		if _, err := os.Stat("/media/data"); os.IsNotExist(err) {
			fmt.Printf("/media/data not found, falling back to %s\n", basePath)
			storagePath = filepath.Join(basePath, opPath)
		}
	} else {
		storagePath = filepath.Join(basePath, opPath)
	}
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		fmt.Printf("Failed to create directory %s: %v\n", storagePath, err)
	}

	return storagePath
}

// tell if we're amd64 or arm64
func getArchitecture() string {
	switch runtime.GOARCH {
	case "arm64", "aarch64":
		return "arm64"
	default:
		return "amd64"
	}
}

func getBasePath() string {
	switch os.Getenv("GS_BASE_PATH") {
	case "":
		return "/opt/nativeplanet/groundseg"
	default:
		return os.Getenv("GS_BASE_PATH")
	}
}

// listen for events from imported packages
func ConfChannel() {
	for {
		event := <-system.ConfChannel
		switch event {
		case "c2cInterval":
			conf := Conf()
			if conf.C2cInterval == 0 {
				if err := UpdateConf(map[string]interface{}{
					"c2cInterval": 600,
				}); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't set C2C interval: %v", err))
				}
			}
		}
	}
}

// update by passing in a map of key:values you want to modify
func UpdateConf(values map[string]interface{}) error {
	// mutex lock to avoid race conditions
	confMutex.Lock()
	defer confMutex.Unlock()
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("Unable to load config: %v", err)
	}
	// unmarshal the config to struct
	var configMap map[string]interface{}
	if err := json.Unmarshal(file, &configMap); err != nil {
		return fmt.Errorf("Error decoding JSON: %v", err)
	}
	// update our unmarshaled struct
	for key, value := range values {
		configMap[key] = value
	}
	if err = persistConf(configMap); err != nil {
		return fmt.Errorf("Unable to persist config update: %v", err)
	}
	return nil
}

func persistConf(configMap map[string]interface{}) error {
	BasePath := getBasePath()
	confPath := filepath.Join(BasePath, "settings", "system.json")
	tmpFile, err := os.CreateTemp(filepath.Dir(confPath), "system.json.*")
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	updatedJSON, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("error encoding JSON: %v", err)
	}
	// write to temp file and validate before overwriting
	if _, err := tmpFile.Write(updatedJSON); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error writing temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file: %v", err)
	}
	fi, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("error checking temp file: %v", err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty configuration file")
	}
	if err := json.Unmarshal(updatedJSON, &globalConfig); err != nil {
		return fmt.Errorf("error updating global config: %v", err)
	}
	if err := os.Rename(tmpPath, confPath); err != nil {
		return fmt.Errorf("error moving temp file: %v", err)
	}
	return nil
}

// we keep map[string]structs.ContainerState in memory to keep track of the containers
// eg if they're running and whether they should be

// modify the desired/actual state of containers
func UpdateContainerState(name string, containerState structs.ContainerState) {
	contMutex.Lock()
	GSContainers[name] = containerState
	logMsg := "<hidden>"
	res, _ := json.Marshal(containerState)
	contMutex.Unlock()
	zap.L().Info(fmt.Sprintf("%s state:%s", name, logMsg))
	zap.L().Debug(fmt.Sprintf("%s state:%s", name, string(res)))
}

// delete a container from the config map
func DeleteContainerState(name string) {
	contMutex.Lock()
	delete(GSContainers, name)
	contMutex.Unlock()
	zap.L().Debug(fmt.Sprintf("%s removed from container state map", name))
}

// get the current container state
func GetContainerState() map[string]structs.ContainerState {
	contMutex.Lock()
	defer contMutex.Unlock()
	return GSContainers
}

// write a default conf to disk
func createDefaultConf() error {
	defaultConfig := defaults.SysConfig(BasePath)
	path := filepath.Join(BasePath, "settings", "system.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&defaultConfig); err != nil {
		return err
	}
	return nil
}

// check outbound tcp connectivity
// takes ip:port
func NetCheck(netCheck string) bool {
	internet := false
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", netCheck, timeout)
	if err != nil {
		errmsg := fmt.Sprintf("Check internet access error: %v", err)
		zap.L().Error(errmsg)
	} else {
		internet = true
		_ = conn.Close()
	}
	return internet
}

// generates a random secret string of the input length
func RandString(length int) string {
	randBytes := make([]byte, length)
	_, err := rand.Read(randBytes)
	if err != nil {
		zap.L().Warn("Random error :s")
		return ""
	}
	return base64.URLEncoding.EncodeToString(randBytes)
}

func GetSHA256(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a new SHA256 hash
	hasher := sha256.New()

	// Copy the file content to the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Get the final hash value
	hashValue := hasher.Sum(nil)

	// Convert the hash to a hexadecimal string
	hashString := hex.EncodeToString(hashValue)

	return hashString, nil
}

// MergeConfigs merges a default and a custom SysConfig, with custom values taking precedence.
func mergeConfigs(defaultConfig, customConfig structs.SysConfig) structs.SysConfig {
	mergedConfig := structs.SysConfig{}

	// GracefulExit
	mergedConfig.GracefulExit = customConfig.GracefulExit || defaultConfig.GracefulExit

	//LastKnownMDNS
	if customConfig.LastKnownMDNS != "" {
		mergedConfig.LastKnownMDNS = customConfig.LastKnownMDNS
	} else {
		mergedConfig.LastKnownMDNS = defaultConfig.LastKnownMDNS
	}

	// Setup

	// if pwhash is empty:
	//    "setup" is "start" (new install)
	// if pwhash not empty:
	// 		if setup is empty:
	//			"setup" is "complete" (migration case)
	//    if setup not empty:
	//      "setup" remains (standard)
	if customConfig.PwHash == "" {
		mergedConfig.Setup = "start"       // new install
		mergedConfig.Salt = RandString(32) // reset salt
	} else {
		if customConfig.Setup == "" {
			mergedConfig.Setup = "complete" // migration case
		} else {
			mergedConfig.Setup = customConfig.Setup // standard
		}
	}

	// EndpointUrl
	if customConfig.EndpointUrl != "" {
		mergedConfig.EndpointUrl = customConfig.EndpointUrl
	} else {
		mergedConfig.EndpointUrl = defaultConfig.EndpointUrl
	}

	// ApiVersion
	if customConfig.ApiVersion != "" {
		mergedConfig.ApiVersion = customConfig.ApiVersion
	} else {
		mergedConfig.ApiVersion = defaultConfig.ApiVersion
	}

	// Piers
	if len(customConfig.Piers) > 0 {
		mergedConfig.Piers = customConfig.Piers
	} else {
		mergedConfig.Piers = defaultConfig.Piers
	}

	// NetCheck
	if customConfig.NetCheck != "" {
		mergedConfig.NetCheck = customConfig.NetCheck
	} else {
		mergedConfig.NetCheck = defaultConfig.NetCheck
	}

	// UpdateMode
	if customConfig.UpdateMode != "" {
		mergedConfig.UpdateMode = customConfig.UpdateMode
	} else {
		mergedConfig.UpdateMode = defaultConfig.UpdateMode
	}

	// UpdateUrl
	if customConfig.UpdateUrl != "" {
		mergedConfig.UpdateUrl = customConfig.UpdateUrl
	} else {
		mergedConfig.UpdateUrl = defaultConfig.UpdateUrl
	}

	// UpdateBranch
	if customConfig.UpdateBranch != "" {
		mergedConfig.UpdateBranch = customConfig.UpdateBranch
	} else {
		mergedConfig.UpdateBranch = defaultConfig.UpdateBranch
	}

	// SwapVal
	if customConfig.SwapVal != 0 {
		mergedConfig.SwapVal = customConfig.SwapVal
	} else {
		mergedConfig.SwapVal = defaultConfig.SwapVal
	}

	// SwapFile
	if customConfig.SwapFile != "" {
		mergedConfig.SwapFile = customConfig.SwapFile
	} else {
		mergedConfig.SwapFile = defaultConfig.SwapFile
	}

	// KeyFile
	if customConfig.KeyFile != "" {
		mergedConfig.KeyFile = customConfig.KeyFile
	} else {
		mergedConfig.KeyFile = defaultConfig.KeyFile
	}

	// Sessions
	if customConfig.Sessions.Authorized != nil {
		mergedConfig.Sessions.Authorized = customConfig.Sessions.Authorized
	} else {
		mergedConfig.Sessions.Authorized = defaultConfig.Sessions.Authorized
	}

	if customConfig.Sessions.Unauthorized != nil {
		mergedConfig.Sessions.Unauthorized = customConfig.Sessions.Unauthorized
	} else {
		mergedConfig.Sessions.Unauthorized = defaultConfig.Sessions.Unauthorized
	}

	// LinuxUpdates
	if customConfig.LinuxUpdates.Value != 0 {
		mergedConfig.LinuxUpdates.Value = customConfig.LinuxUpdates.Value
	} else {
		mergedConfig.LinuxUpdates.Value = defaultConfig.LinuxUpdates.Value
	}

	if customConfig.LinuxUpdates.Interval != "" {
		mergedConfig.LinuxUpdates.Interval = customConfig.LinuxUpdates.Interval
	} else {
		mergedConfig.LinuxUpdates.Interval = defaultConfig.LinuxUpdates.Interval
	}
	// DockerData
	if customConfig.DockerData != "" {
		mergedConfig.DockerData = customConfig.DockerData
	} else {
		mergedConfig.DockerData = defaultConfig.DockerData
	}

	// WgOn
	mergedConfig.WgOn = customConfig.WgOn || defaultConfig.WgOn

	// WgRegistered
	mergedConfig.WgRegistered = customConfig.WgRegistered || defaultConfig.WgRegistered

	// StartramSetReminder
	mergedConfig.StartramSetReminder.One = customConfig.StartramSetReminder.One || defaultConfig.StartramSetReminder.One
	mergedConfig.StartramSetReminder.Three = customConfig.StartramSetReminder.Three || defaultConfig.StartramSetReminder.Three
	mergedConfig.StartramSetReminder.Seven = customConfig.StartramSetReminder.Seven || defaultConfig.StartramSetReminder.Seven

	// DiskWarning
	mergedConfig.DiskWarning = customConfig.DiskWarning

	// PwHash
	if customConfig.PwHash != "" {
		mergedConfig.PwHash = customConfig.PwHash
	} else {
		mergedConfig.PwHash = defaultConfig.PwHash
	}

	// C2cInterval
	if customConfig.C2cInterval != 0 {
		mergedConfig.C2cInterval = customConfig.C2cInterval
	} else {
		mergedConfig.C2cInterval = defaultConfig.C2cInterval
	}

	// GsVersion
	if customConfig.GsVersion != "" {
		mergedConfig.GsVersion = customConfig.GsVersion
	} else {
		mergedConfig.GsVersion = defaultConfig.GsVersion
	}

	// CfgDir
	if customConfig.CfgDir != "" {
		mergedConfig.CfgDir = customConfig.CfgDir
	} else {
		mergedConfig.CfgDir = defaultConfig.CfgDir
	}

	// UpdateInterval
	if customConfig.UpdateInterval != 0 {
		mergedConfig.UpdateInterval = customConfig.UpdateInterval
	} else {
		mergedConfig.UpdateInterval = defaultConfig.UpdateInterval
	}

	// BinHash
	if customConfig.BinHash != "" {
		mergedConfig.BinHash = customConfig.BinHash
	} else {
		mergedConfig.BinHash = defaultConfig.BinHash
	}

	// Pubkey
	if customConfig.Pubkey != "" {
		mergedConfig.Pubkey = customConfig.Pubkey
	} else {
		mergedConfig.Pubkey = defaultConfig.Pubkey
	}

	// Privkey
	if customConfig.Privkey != "" {
		mergedConfig.Privkey = customConfig.Privkey
	} else {
		mergedConfig.Privkey = defaultConfig.Privkey
	}

	// Salt
	if mergedConfig.Salt == "" {
		mergedConfig.Salt = customConfig.Salt
	}

	// PenpaiAllow
	mergedConfig.PenpaiAllow = customConfig.PenpaiAllow || defaultConfig.PenpaiAllow

	// PenpaiCores
	if customConfig.PenpaiCores != 0 {
		mergedConfig.PenpaiCores = customConfig.PenpaiCores
	} else {
		mergedConfig.PenpaiCores = defaultConfig.PenpaiCores
	}

	// PenpaiModels
	// always use defaults as newest
	mergedConfig.PenpaiModels = defaultConfig.PenpaiModels

	// PenpaiRunning
	mergedConfig.PenpaiRunning = customConfig.PenpaiRunning

	// PenpaiActive
	validModel := false
	for _, model := range defaultConfig.PenpaiModels {
		if strings.EqualFold(model.ModelName, customConfig.PenpaiActive) {
			validModel = true
		}
	}
	if customConfig.PenpaiActive != "" && validModel {
		mergedConfig.PenpaiActive = customConfig.PenpaiActive
	} else {
		mergedConfig.PenpaiActive = defaultConfig.PenpaiActive
	}

	// 502 checker
	if customConfig.Disable502 {
		mergedConfig.Disable502 = customConfig.Disable502
	} else {
		mergedConfig.Disable502 = defaultConfig.Disable502
	}
	mergedConfig.RemoteBackupPassword = customConfig.RemoteBackupPassword
	if customConfig.SnapTime == 0 {
		mergedConfig.SnapTime = defaultConfig.SnapTime
	} else {
		mergedConfig.SnapTime = customConfig.SnapTime
	}
	return mergedConfig
}

func setDebugMode() {
	for _, arg := range os.Args[1:] {
		// trigger this with `./groundseg dev`
		if arg == "dev" {
			zap.L().Info("Starting GroundSeg in debug mode")
			DebugMode = true
		}
	}
}

func configureFixerScript() {
	if err := system.FixerScript(BasePath); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to configure fixer script: %v", err))
	}
}
