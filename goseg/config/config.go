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
	// cached /retrieve blob
	startramConfig        structs.StartramRetrieve
	startramConfigUpdated time.Time
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
	contMutex     sync.RWMutex
	versMutex     sync.Mutex
	minioPwdMutex sync.Mutex
	startramMu    sync.RWMutex
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
			return
		}
		file, err = os.Open(confPath)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to open config after creation: %v", err))
			return
		}
		salt := RandString(32)
		wgPriv, wgPub, err := WgKeyGen()
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		} else {
			if err = UpdateConfTyped(
				WithPubkey(wgPub),
				WithPrivkey(wgPriv),
				WithSalt(salt),
				WithKeyfile(keyPath),
			); err != nil {
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
	if err := file.Close(); err != nil {
		zap.L().Warn(fmt.Sprintf("Error closing JSON before reload: %v", err))
	}
	file, err = os.Open(confPath)
	if err != nil {
		errmsg := fmt.Sprintf("Error opening JSON: %v", err)
		zap.L().Error(errmsg)
		return
	}
	defer file.Close()
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
		if err = UpdateConfTyped(WithKeyfile(keyPath)); err != nil {
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

type StartramConfigSnapshot struct {
	Value     structs.StartramRetrieve
	UpdatedAt time.Time
	Fresh     bool
}

type StartramSettings struct {
	EndpointURL          string
	Pubkey               string
	RemoteBackupPassword string
	WgRegistered         bool
	WgOn                 bool
	Piers                []string
}

type AuthSettings struct {
	KeyFile            string
	Salt               string
	PasswordHash       string
	AuthorizedSessions map[string]structs.SessionInfo
}

type PenpaiSettings struct {
	Models      []structs.Penpai
	Allowed     bool
	ActiveModel string
	Running     bool
	ActiveCores int
}

type Check502Settings struct {
	Piers      []string
	WgOn       bool
	Disable502 bool
}

type ShipSettings struct {
	Piers []string
}

type ConnectivitySettings struct {
	C2cInterval int
}

type UpdateSettings struct {
	UpdateMode   string
	UpdateBranch string
}

type SwapSettings struct {
	SwapFile string
	SwapVal  int
}

type ShipRuntimeSettings struct {
	SnapTime int
}

type RuntimeSettings struct {
	BasePath     string
	Architecture string
	DebugMode    bool
}

func SetStartramConfig(retrieve structs.StartramRetrieve) {
	startramMu.Lock()
	defer startramMu.Unlock()
	startramConfig = retrieve
	startramConfigUpdated = time.Now()
}

func GetStartramConfig() structs.StartramRetrieve {
	startramMu.RLock()
	defer startramMu.RUnlock()
	return startramConfig
}

func StartramSettingsSnapshot() StartramSettings {
	conf := Conf()
	return StartramSettings{
		EndpointURL:          conf.EndpointUrl,
		Pubkey:               conf.Pubkey,
		RemoteBackupPassword: conf.RemoteBackupPassword,
		WgRegistered:         conf.WgRegistered,
		WgOn:                 conf.WgOn,
		Piers:                append([]string(nil), conf.Piers...),
	}
}

func AuthSettingsSnapshot() AuthSettings {
	conf := Conf()
	authorizedSessions := make(map[string]structs.SessionInfo, len(conf.Sessions.Authorized))
	for tokenID, session := range conf.Sessions.Authorized {
		authorizedSessions[tokenID] = session
	}
	return AuthSettings{
		KeyFile:            conf.KeyFile,
		Salt:               conf.Salt,
		PasswordHash:       conf.PwHash,
		AuthorizedSessions: authorizedSessions,
	}
}

func PenpaiSettingsSnapshot() PenpaiSettings {
	conf := Conf()
	return PenpaiSettings{
		Models:      append([]structs.Penpai(nil), conf.PenpaiModels...),
		Allowed:     conf.PenpaiAllow,
		ActiveModel: conf.PenpaiActive,
		Running:     conf.PenpaiRunning,
		ActiveCores: conf.PenpaiCores,
	}
}

func Check502SettingsSnapshot() Check502Settings {
	conf := Conf()
	return Check502Settings{
		Piers:      append([]string(nil), conf.Piers...),
		WgOn:       conf.WgOn,
		Disable502: conf.Disable502,
	}
}

func ShipSettingsSnapshot() ShipSettings {
	conf := Conf()
	return ShipSettings{
		Piers: append([]string(nil), conf.Piers...),
	}
}

func ConnectivitySettingsSnapshot() ConnectivitySettings {
	conf := Conf()
	return ConnectivitySettings{
		C2cInterval: conf.C2cInterval,
	}
}

func UpdateSettingsSnapshot() UpdateSettings {
	conf := Conf()
	return UpdateSettings{
		UpdateMode:   conf.UpdateMode,
		UpdateBranch: conf.UpdateBranch,
	}
}

func SwapSettingsSnapshot() SwapSettings {
	conf := Conf()
	return SwapSettings{
		SwapFile: conf.SwapFile,
		SwapVal:  conf.SwapVal,
	}
}

func ShipRuntimeSettingsSnapshot() ShipRuntimeSettings {
	conf := Conf()
	return ShipRuntimeSettings{
		SnapTime: conf.SnapTime,
	}
}

func RuntimeSettingsSnapshot() RuntimeSettings {
	return RuntimeSettings{
		BasePath:     BasePath,
		Architecture: Architecture,
		DebugMode:    DebugMode,
	}
}

func GetStartramConfigSnapshot() StartramConfigSnapshot {
	startramMu.RLock()
	defer startramMu.RUnlock()
	return StartramConfigSnapshot{
		Value:     startramConfig,
		UpdatedAt: startramConfigUpdated,
		Fresh:     !startramConfigUpdated.IsZero(),
	}
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
				if err := UpdateConfTyped(WithC2cInterval(600)); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't set C2C interval: %v", err))
				}
			}
		}
	}
}

type ConfUpdateOption func(*ConfPatch)

type ConfPatch struct {
	Piers                 *[]string
	WgOn                  *bool
	StartramReminderOne   *bool
	StartramReminderThree *bool
	StartramReminderSeven *bool
	PenpaiAllow           *bool
	GracefulExit          *bool
	SwapVal               *int
	PenpaiRunning         *bool
	PenpaiActive          *string
	PenpaiCores           *int
	EndpointURL           *string
	WgRegistered          *bool
	RemoteBackupPassword  *string
	Pubkey                *string
	Privkey               *string
	Salt                  *string
	KeyFile               *string
	C2cInterval           *int
	Setup                 *string
	PwHash                *string
	AuthorizedSessions    map[string]structs.SessionInfo
	UnauthorizedSessions  map[string]structs.SessionInfo
	DiskWarning           *map[string]structs.DiskWarning
	LastKnownMDNS         *string
	DisableSlsa           *bool
	GSVersion             *string
	BinHash               *string
}

func (patch *ConfPatch) hasUpdates() bool {
	return patch.Piers != nil ||
		patch.WgOn != nil ||
		patch.StartramReminderOne != nil ||
		patch.StartramReminderThree != nil ||
		patch.StartramReminderSeven != nil ||
		patch.PenpaiAllow != nil ||
		patch.GracefulExit != nil ||
		patch.SwapVal != nil ||
		patch.PenpaiRunning != nil ||
		patch.PenpaiActive != nil ||
		patch.PenpaiCores != nil ||
		patch.EndpointURL != nil ||
		patch.WgRegistered != nil ||
		patch.RemoteBackupPassword != nil ||
		patch.Pubkey != nil ||
		patch.Privkey != nil ||
		patch.Salt != nil ||
		patch.KeyFile != nil ||
		patch.C2cInterval != nil ||
		patch.Setup != nil ||
		patch.PwHash != nil ||
		len(patch.AuthorizedSessions) > 0 ||
		len(patch.UnauthorizedSessions) > 0 ||
		patch.DiskWarning != nil ||
		patch.LastKnownMDNS != nil ||
		patch.DisableSlsa != nil ||
		patch.GSVersion != nil ||
		patch.BinHash != nil
}

// UpdateConfTyped applies typed mutations to SysConfig and persists them.
func UpdateConfTyped(opts ...ConfUpdateOption) error {
	if len(opts) == 0 {
		return nil
	}

	patch := &ConfPatch{}
	for _, opt := range opts {
		if opt != nil {
			opt(patch)
		}
	}
	if !patch.hasUpdates() {
		return nil
	}

	return updateConfFromPatch(patch)
}

func updateConfFromPatch(patch *ConfPatch) error {
	confMutex.Lock()
	defer confMutex.Unlock()

	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("Unable to load config: %v", err)
	}

	var configStruct structs.SysConfig
	if err := json.Unmarshal(file, &configStruct); err != nil {
		return fmt.Errorf("Error decoding JSON: %v", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(file, &configMap); err != nil {
		return fmt.Errorf("Error decoding JSON map: %v", err)
	}

	applyConfPatch(&configStruct, patch)

	typedMap, err := structToMap(configStruct)
	if err != nil {
		return fmt.Errorf("Error encoding typed config map: %v", err)
	}
	for key, value := range typedMap {
		configMap[key] = value
	}

	if err := persistConf(configMap); err != nil {
		return fmt.Errorf("Unable to persist config update: %v", err)
	}
	return nil
}

func applyConfPatch(configStruct *structs.SysConfig, patch *ConfPatch) {
	if patch.Piers != nil {
		configStruct.Piers = append([]string(nil), (*patch.Piers)...)
	}
	if patch.WgOn != nil {
		configStruct.WgOn = *patch.WgOn
	}
	if patch.StartramReminderOne != nil {
		configStruct.StartramSetReminder.One = *patch.StartramReminderOne
	}
	if patch.StartramReminderThree != nil {
		configStruct.StartramSetReminder.Three = *patch.StartramReminderThree
	}
	if patch.StartramReminderSeven != nil {
		configStruct.StartramSetReminder.Seven = *patch.StartramReminderSeven
	}
	if patch.PenpaiAllow != nil {
		configStruct.PenpaiAllow = *patch.PenpaiAllow
	}
	if patch.GracefulExit != nil {
		configStruct.GracefulExit = *patch.GracefulExit
	}
	if patch.SwapVal != nil {
		configStruct.SwapVal = *patch.SwapVal
	}
	if patch.PenpaiRunning != nil {
		configStruct.PenpaiRunning = *patch.PenpaiRunning
	}
	if patch.PenpaiActive != nil {
		configStruct.PenpaiActive = *patch.PenpaiActive
	}
	if patch.PenpaiCores != nil {
		configStruct.PenpaiCores = *patch.PenpaiCores
	}
	if patch.EndpointURL != nil {
		configStruct.EndpointUrl = *patch.EndpointURL
	}
	if patch.WgRegistered != nil {
		configStruct.WgRegistered = *patch.WgRegistered
	}
	if patch.RemoteBackupPassword != nil {
		configStruct.RemoteBackupPassword = *patch.RemoteBackupPassword
	}
	if patch.Pubkey != nil {
		configStruct.Pubkey = *patch.Pubkey
	}
	if patch.Privkey != nil {
		configStruct.Privkey = *patch.Privkey
	}
	if patch.Salt != nil {
		configStruct.Salt = *patch.Salt
	}
	if patch.KeyFile != nil {
		configStruct.KeyFile = *patch.KeyFile
	}
	if patch.C2cInterval != nil {
		configStruct.C2cInterval = *patch.C2cInterval
	}
	if patch.Setup != nil {
		configStruct.Setup = *patch.Setup
	}
	if patch.PwHash != nil {
		configStruct.PwHash = *patch.PwHash
	}
	if len(patch.AuthorizedSessions) > 0 {
		if configStruct.Sessions.Authorized == nil {
			configStruct.Sessions.Authorized = make(map[string]structs.SessionInfo)
		}
		for tokenID, session := range patch.AuthorizedSessions {
			configStruct.Sessions.Authorized[tokenID] = session
		}
	}
	if len(patch.UnauthorizedSessions) > 0 {
		if configStruct.Sessions.Unauthorized == nil {
			configStruct.Sessions.Unauthorized = make(map[string]structs.SessionInfo)
		}
		for tokenID, session := range patch.UnauthorizedSessions {
			configStruct.Sessions.Unauthorized[tokenID] = session
		}
	}
	if patch.DiskWarning != nil {
		copied := make(map[string]structs.DiskWarning, len(*patch.DiskWarning))
		for key, warning := range *patch.DiskWarning {
			copied[key] = warning
		}
		configStruct.DiskWarning = copied
	}
	if patch.LastKnownMDNS != nil {
		configStruct.LastKnownMDNS = *patch.LastKnownMDNS
	}
	if patch.DisableSlsa != nil {
		configStruct.DisableSlsa = *patch.DisableSlsa
	}
	if patch.GSVersion != nil {
		configStruct.GsVersion = *patch.GSVersion
	}
	if patch.BinHash != nil {
		configStruct.BinHash = *patch.BinHash
	}
}

func structToMap(configStruct structs.SysConfig) (map[string]interface{}, error) {
	configBytes, err := json.Marshal(configStruct)
	if err != nil {
		return nil, err
	}
	configMap := make(map[string]interface{})
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return nil, err
	}
	return configMap, nil
}

func WithPiers(piers []string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		copied := append([]string(nil), piers...)
		patch.Piers = &copied
	}
}

func WithWgOn(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.WgOn = &enabled
	}
}

func WithStartramReminderOne(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderOne = &reminded
	}
}

func WithStartramReminderThree(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderThree = &reminded
	}
}

func WithStartramReminderSeven(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderSeven = &reminded
	}
}

func WithStartramReminderAll(reminded bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.StartramReminderOne = &reminded
		patch.StartramReminderThree = &reminded
		patch.StartramReminderSeven = &reminded
	}
}

func WithPenpaiAllow(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiAllow = &enabled
	}
}

func WithGracefulExit(enabled bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.GracefulExit = &enabled
	}
}

func WithSwapVal(value int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.SwapVal = &value
	}
}

func WithPenpaiRunning(running bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiRunning = &running
	}
}

func WithPenpaiActive(model string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiActive = &model
	}
}

func WithPenpaiCores(cores int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PenpaiCores = &cores
	}
}

func WithEndpointURL(endpoint string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.EndpointURL = &endpoint
	}
}

func WithWgRegistered(registered bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.WgRegistered = &registered
	}
}

func WithRemoteBackupPassword(password string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.RemoteBackupPassword = &password
	}
}

func WithPubkey(pubkey string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Pubkey = &pubkey
	}
}

func WithPrivkey(privkey string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Privkey = &privkey
	}
}

func WithSalt(salt string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Salt = &salt
	}
}

func WithKeyfile(path string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.KeyFile = &path
	}
}

func WithC2cInterval(seconds int) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.C2cInterval = &seconds
	}
}

func WithSetup(step string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.Setup = &step
	}
}

func WithPwHash(hash string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.PwHash = &hash
	}
}

func WithAuthorizedSession(tokenID string, session structs.SessionInfo) ConfUpdateOption {
	return func(patch *ConfPatch) {
		if patch.AuthorizedSessions == nil {
			patch.AuthorizedSessions = make(map[string]structs.SessionInfo)
		}
		patch.AuthorizedSessions[tokenID] = session
	}
}

func WithUnauthorizedSession(tokenID string, session structs.SessionInfo) ConfUpdateOption {
	return func(patch *ConfPatch) {
		if patch.UnauthorizedSessions == nil {
			patch.UnauthorizedSessions = make(map[string]structs.SessionInfo)
		}
		patch.UnauthorizedSessions[tokenID] = session
	}
}

func WithDiskWarning(warning map[string]structs.DiskWarning) ConfUpdateOption {
	return func(patch *ConfPatch) {
		copied := make(map[string]structs.DiskWarning, len(warning))
		for key, val := range warning {
			copied[key] = val
		}
		patch.DiskWarning = &copied
	}
}

func WithLastKnownMDNS(url string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.LastKnownMDNS = &url
	}
}

func WithDisableSlsa(disable bool) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.DisableSlsa = &disable
	}
}

func WithGSVersion(version string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.GSVersion = &version
	}
}

func WithBinHash(hash string) ConfUpdateOption {
	return func(patch *ConfPatch) {
		patch.BinHash = &hash
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
	contMutex.RLock()
	defer contMutex.RUnlock()
	stateCopy := make(map[string]structs.ContainerState, len(GSContainers))
	for name, state := range GSContainers {
		stateCopy[name] = state
	}
	return stateCopy
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
