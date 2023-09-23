package config

// code for managing groundseg and container configurations

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goseg/defaults"
	"goseg/logger"
	"goseg/structs"
	"goseg/system"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	// global settings config (accessed via funcs)
	globalConfig structs.SysConfig
	// base path for installation (override default with env var)
	// BasePath = os.Getenv("GS_BASE_PATH")
	BasePath = "/opt/nativeplanet/groundseg"
	// only amd64 or arm64
	Architecture = getArchitecture()
	// struct of /retrieve blob
	StartramConfig structs.StartramRetrieve
	// struct of minio passwords
	minIOPasswords = make(map[string]string)
	// unused for now, set with `./groundseg dev`
	DebugMode = false
	Ready     = false
	// representation of desired/actual container states
	GSContainers = make(map[string]structs.ContainerState)
	// channel for log stream requests
	LogsEventBus  = make(chan structs.LogsEvent, 100)
	DockerDir     = "/var/lib/docker/volumes/"
	confPath      = filepath.Join(BasePath, "settings", "system.json")
	confMutex     sync.Mutex
	contMutex     sync.Mutex
	versMutex     sync.Mutex
	minioPwdMutex sync.Mutex
)

// try initializing from system.json on disk
func init() {
	logger.Logger.Info("Starting GroundSeg")
	logger.Logger.Info("Urbit is love <3")
	for _, arg := range os.Args[1:] {
		// trigger this with `./groundseg dev`
		if arg == "dev" {
			logger.Logger.Info("Starting GroundSeg in debug mode")
			DebugMode = true
		}
	}
	if BasePath == "" {
		// default base path
		BasePath = "/opt/nativeplanet/groundseg"
	}
	if err := system.FixerScript(BasePath); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Unable to configure fixer script: %v", err))
	}
	logger.Logger.Info(fmt.Sprintf("Loading configs from %s", BasePath))
	confPath := filepath.Join(BasePath, "settings", "system.json")
	file, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = createDefaultConf()
		if err != nil {
			// panic if we can't create it
			logger.Logger.Error(fmt.Sprintf("Unable to create config! %v", err))
			fmt.Println(fmt.Sprintf("Failed to create log directory: %v", err))
			fmt.Println("\n\n.・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.")
			fmt.Println("Please run GroundSeg as root!  \n /) /)\n( . . )\n(  >< )\n Love, Native Planet")
			fmt.Println(".・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n\n")
			panic("")
		}
		// generate and insert wireguard keys
		wgPriv, wgPub, err := WgKeyGen()
		salt := RandString(32)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
		} else {
			err = UpdateConf(map[string]interface{}{
				"pubkey":  wgPub,
				"privkey": wgPriv,
				"salt":    salt,
			})
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
			}
		}
		keyPath := filepath.Join(BasePath, "settings", "session.key")
		keyfile, err := os.Stat(keyPath)
		if err != nil || keyfile.Size() == 0 {
			keyContent := RandString(32)
			if err := ioutil.WriteFile(keyPath, []byte(keyContent), 0644); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't write keyfile! %v", err))
			}
		}
		file, _ = os.Open(confPath)
	}
	defer file.Close()
	// read the sysconfig to memory
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&globalConfig); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error decoding JSON: %v", err))
	}
	// wipe the sessions on each startup
	//globalConfig.Sessions.Authorized = make(map[string]structs.SessionInfo)
	globalConfig.Sessions.Unauthorized = make(map[string]structs.SessionInfo)

	// get hash of groundseg binary
	hash, err := getSHA256(filepath.Join(BasePath, "groundseg"))
	if err != nil {
		errmsg := fmt.Sprintf("Error getting binary sha256 hash: %v", err)
		logger.Logger.Error(errmsg)
	}
	globalConfig.BinHash = hash
	logger.Logger.Info(fmt.Sprintf("Binary sha256 hash: %v", hash))

	configMap := make(map[string]interface{})
	configBytes, err := json.Marshal(globalConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshaling JSON: %v", err)
		logger.Logger.Error(errmsg)
	}
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshaling JSON: %v", err)
		logger.Logger.Error(errmsg)
	}
	err = persistConf(configMap)
	if err != nil {
		errmsg := fmt.Sprintf("Error persisting JSON: %v", err)
		logger.Logger.Error(errmsg)
	}
	file, err = os.Open(confPath)
	if err != nil {
		errmsg := fmt.Sprintf("Error opening JSON: %v", err)
		logger.Logger.Error(errmsg)
	}
	decoder = json.NewDecoder(file)
	err = decoder.Decode(&globalConfig)
	if err != nil {
		errmsg := fmt.Sprintf("Error decoding JSON: %v", err)
		logger.Logger.Error(errmsg)
	}
}

// return the global conf var
func Conf() structs.SysConfig {
	confMutex.Lock()
	defer confMutex.Unlock()
	return globalConfig
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
	if BasePath == "" {
		// default base path
		BasePath = "/opt/nativeplanet/groundseg"
	}
	// marshal and persist it
	updatedJSON, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("Error encoding JSON: %v", err)
	}
	// update the globalConfig var
	if err := json.Unmarshal(updatedJSON, &globalConfig); err != nil {
		return fmt.Errorf("Error updating global config: %v", err)
	}
	// write to disk
	if err := ioutil.WriteFile(confPath, updatedJSON, 0644); err != nil {
		return fmt.Errorf("Error writing to file: %v", err)
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
	logger.Logger.Info(fmt.Sprintf("%s state:%s", name, logMsg))
	logger.Logger.Debug(fmt.Sprintf("%s state:%s", name, string(res)))
}

// delete a container from the config map
func DeleteContainerState(name string) {
	contMutex.Lock()
	delete(GSContainers, name)
	contMutex.Unlock()
	logger.Logger.Debug(fmt.Sprintf("%s removed from container state map", name))
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
		logger.Logger.Error(errmsg)
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
		logger.Logger.Warn("Random error :s")
		return ""
	}
	return base64.URLEncoding.EncodeToString(randBytes)
}

func getSHA256(filePath string) (string, error) {
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
