package config

import (
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/httpx"
	"groundseg/structs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type versionHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type versionRuntime struct {
	stateMu                sync.RWMutex
	opsMu                  sync.Mutex
	versionStore           VersionStore
	versionHTTPClient      versionHTTPDoer
	versionFetchRetryCount int
	versionFetchRetryDelay time.Duration
	versionFetchSleep      func(time.Duration)
}

var versionRuntimeState = &versionRuntime{
	versionStore:           newInMemoryVersionStore(),
	versionHTTPClient:      http.DefaultClient,
	versionFetchRetryCount: 10,
	versionFetchRetryDelay: time.Second,
	versionFetchSleep:      time.Sleep,
}

type VersionState struct {
	Channel     structs.Channel
	ServerReady bool
}

type VersionStore interface {
	Snapshot() VersionState
	SetState(channel structs.Channel, ready bool)
	SetChannel(channel structs.Channel)
	SetServerReady(ready bool)
}

type inMemoryVersionStore struct {
	mu          sync.RWMutex
	channel     structs.Channel
	serverReady bool
}

func newInMemoryVersionStore() *inMemoryVersionStore {
	return &inMemoryVersionStore{}
}

func (store *inMemoryVersionStore) Snapshot() VersionState {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return VersionState{
		Channel:     store.channel,
		ServerReady: store.serverReady,
	}
}

func (store *inMemoryVersionStore) SetState(channel structs.Channel, ready bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.channel = channel
	store.serverReady = ready
}

func (store *inMemoryVersionStore) SetChannel(channel structs.Channel) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.channel = channel
}

func (store *inMemoryVersionStore) SetServerReady(ready bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.serverReady = ready
}

func versionRuntimeSnapshot() *versionRuntime {
	return versionRuntimeState
}

func setVersionStore(store VersionStore) {
	if store == nil {
		return
	}
	state := versionRuntimeSnapshot()
	state.stateMu.Lock()
	defer state.stateMu.Unlock()
	state.versionStore = store
}

func versionStoreSnapshot() VersionStore {
	state := versionRuntimeSnapshot()
	state.stateMu.RLock()
	defer state.stateMu.RUnlock()
	return state.versionStore
}

func setVersionHTTPClient(client versionHTTPDoer) {
	if client == nil {
		return
	}
	state := versionRuntimeSnapshot()
	state.stateMu.Lock()
	defer state.stateMu.Unlock()
	state.versionHTTPClient = client
}

func versionHTTPClientSnapshot() versionHTTPDoer {
	state := versionRuntimeSnapshot()
	state.stateMu.RLock()
	defer state.stateMu.RUnlock()
	return state.versionHTTPClient
}

func setVersionFetchPolicy(retries int, delay time.Duration) {
	state := versionRuntimeSnapshot()
	state.stateMu.Lock()
	defer state.stateMu.Unlock()
	if retries > 0 {
		state.versionFetchRetryCount = retries
	}
	if delay > 0 {
		state.versionFetchRetryDelay = delay
	}
}

func setVersionFetchSleep(sleepFn func(time.Duration)) {
	state := versionRuntimeSnapshot()
	state.stateMu.Lock()
	defer state.stateMu.Unlock()
	state.versionFetchSleep = sleepFn
}

func versionFetchConfigSnapshot() (int, time.Duration, func(time.Duration)) {
	state := versionRuntimeSnapshot()
	state.stateMu.RLock()
	defer state.stateMu.RUnlock()
	sleepFn := state.versionFetchSleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}
	return state.versionFetchRetryCount, state.versionFetchRetryDelay, sleepFn
}

func SetVersionStore(store VersionStore) {
	setVersionStore(store)
}

func SetVersionHTTPClient(client versionHTTPDoer) {
	setVersionHTTPClient(client)
}

func SetVersionFetchRetryPolicy(retries int, delay time.Duration) {
	setVersionFetchPolicy(retries, delay)
}

func GetVersionState() VersionState {
	return versionStoreSnapshot().Snapshot()
}

func GetVersionChannel() structs.Channel {
	return versionStoreSnapshot().Snapshot().Channel
}

func SetVersionChannel(channel structs.Channel) {
	versionStoreSnapshot().SetChannel(channel)
}

func IsVersionServerReady() bool {
	return versionStoreSnapshot().Snapshot().ServerReady
}

func fetchVersionFromServer(conf structs.SysConfig) (structs.Version, error) {
	retries, delay, sleepFn := versionFetchConfigSnapshot()
	if retries < 1 {
		retries = 1
	}
	url := globalConfig.Connectivity.UpdateURL
	client := versionHTTPClientSnapshot()
	if client == nil {
		client = http.DefaultClient
	}
	var fetchedVersion structs.Version
	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return structs.Version{}, err
		}
		userAgent := "NativePlanet.GroundSeg-" + conf.Runtime.GsVersion
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			errmsg := fmt.Sprintf("Unable to connect to update server: %v", err)
			zap.L().Warn(errmsg)
			if i < retries-1 {
				sleepFn(delay)
				continue
			}
			return structs.Version{}, fmt.Errorf("request version metadata: %w", err)
		}

		if err := httpx.ReadJSON(resp, url, &fetchedVersion); err != nil {
			errmsg := fmt.Sprintf("Error decoding version metadata: %v", err)
			zap.L().Warn(errmsg)
			if i < retries-1 {
				sleepFn(delay)
				continue
			}
			return structs.Version{}, fmt.Errorf("decode version metadata: %w", err)
		}

		return fetchedVersion, nil
	}
	return structs.Version{}, fmt.Errorf("version fetch failed after retries")
}

func persistVersionInfo(version structs.Version) error {
	confPath := filepath.Join(BasePath(), "settings", "version_info.json")
	rawVersionInfo, err := json.MarshalIndent(version, "", "    ")
	if err != nil {
		return err
	}
	return persistConfigJSON(confPath, rawVersionInfo)
}

func resolveVersionForConfiguredChannel(version structs.Version, releaseChannel string) (structs.Channel, error) {
	channel, exists := version.Groundseg[releaseChannel]
	if !exists {
		return structs.Channel{}, fmt.Errorf("missing release channel %s", releaseChannel)
	}
	return channel, nil
}

// ResolveLatestChannel fetches remote metadata and resolves the configured channel without mutating state.
func ResolveLatestChannel(conf structs.SysConfig) (structs.Version, structs.Channel, error) {
	fetchedVersion, err := fetchVersionFromServer(conf)
	if err != nil {
		return structs.Version{}, structs.Channel{}, err
	}
	channel, err := resolveVersionForConfiguredChannel(fetchedVersion, conf.Connectivity.UpdateBranch)
	if err != nil {
		return structs.Version{}, structs.Channel{}, err
	}
	return fetchedVersion, channel, nil
}

// PublishVersionMetadata persists release metadata and publishes current channel state.
func PublishVersionMetadata(version structs.Version, channel structs.Channel) error {
	if err := persistVersionInfo(version); err != nil {
		return err
	}
	versionStoreSnapshot().SetState(channel, true)
	return nil
}

// CheckVersionWithError fetches release metadata and returns the channel for the
// current branch.
func CheckVersionWithError() (structs.Channel, error) {
	runtime := versionRuntimeSnapshot()
	runtime.opsMu.Lock()
	defer runtime.opsMu.Unlock()

	conf := Config()
	_, channel, err := ResolveLatestChannel(conf)
	if err != nil {
		return GetVersionChannel(), fmt.Errorf("resolve latest version channel: %w", err)
	}
	return channel, nil
}

// CheckVersion fetches release metadata and returns the channel for the current
// branch. It preserves the legacy boolean contract by adapting the explicit error
// variant.
func CheckVersion() (structs.Channel, bool) {
	channel, err := CheckVersionWithError()
	return channel, err == nil
}

// SyncVersionInfo fetches remote metadata, persists it, and refreshes version globals.
func SyncVersionInfo() (structs.Channel, bool) {
	channel, err := SyncVersionInfoWithError()
	return channel, err == nil
}

// SyncVersionInfoWithError fetches remote metadata, persists it, and refreshes
// version globals, returning the active channel or an error.
func SyncVersionInfoWithError() (structs.Channel, error) {
	runtime := versionRuntimeSnapshot()
	runtime.opsMu.Lock()
	defer runtime.opsMu.Unlock()

	conf := Config()
	fetchedVersion, channel, err := ResolveLatestChannel(conf)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to resolve latest version channel: %v", err))
		versionStoreSnapshot().SetServerReady(false)
		return GetVersionChannel(), fmt.Errorf("resolve latest version channel: %w", err)
	}

	if err := PublishVersionMetadata(fetchedVersion, channel); err != nil {
		errmsg := fmt.Sprintf("Failed to persist version metadata: %v", err)
		zap.L().Error(errmsg)
		versionStoreSnapshot().SetServerReady(false)
		return GetVersionChannel(), fmt.Errorf("publish version metadata: %w", err)
	}
	return GetVersionChannel(), nil
}

// write the defaults.VersionInfo value to disk
func CreateDefaultVersion() error {
	versionInfo, err := defaults.DefaultVersionDefaults()
	if err != nil {
		return err
	}
	rawVersionInfo, err := json.MarshalIndent(versionInfo, "", "    ")
	if err != nil {
		return err
	}
	filePath := filepath.Join(BasePath(), "settings", "version_info.json")
	err = persistConfigJSON(filePath, rawVersionInfo)
	if err != nil {
		return err
	}
	return nil
}

// return the existing local version info or create default
func LocalVersion() structs.Version {
	defaultVersion := func() structs.Version {
		fallback, err := defaults.DefaultVersionDefaults()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to decode embedded default version metadata: %v", err))
		}
		return fallback
	}
	confPath := filepath.Join(BasePath(), "settings", "version_info.json")
	_, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = CreateDefaultVersion()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to write version info! %v", err))
			return defaultVersion()
		}
	}
	file, err := os.ReadFile(confPath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Unable to load version info: %v", err))
		return defaultVersion()
	}
	var versionStruct structs.Version
	if err := json.Unmarshal(file, &versionStruct); err != nil {
		zap.L().Error(fmt.Sprintf("Error decoding version JSON: %v", err))
		return defaultVersion()
	}
	return versionStruct
}
