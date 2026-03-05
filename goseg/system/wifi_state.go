package system

import (
	"context"
	"sync"

	"groundseg/structs"
)

// C2CStoredSSIDs is kept in the Wi-Fi module for C2C orchestration state.
var (
	C2CStoredSSIDs               []string
	defaultWiFiRuntimeStateValue = newWiFiRuntimeState()
)

type wifiRuntimeState struct {
	wifiInfo         structs.SystemWifi
	device           string
	localUrl         string
	confChannel      chan string
	wifiInit         sync.Once
	wifiStateMu      sync.RWMutex
	c2cMu            sync.Mutex
	c2cEnabled       bool
	wifiInfoLoopStop context.CancelFunc
	stopMu           sync.Mutex
}

func DefaultWiFiRuntimeState() *wifiRuntimeState {
	return defaultWiFiRuntimeStateValue
}

func newWiFiRuntimeState() *wifiRuntimeState {
	return &wifiRuntimeState{
		confChannel: make(chan string, 100),
	}
}

func resolveWiFiRuntimeState(overrides ...*wifiRuntimeState) *wifiRuntimeState {
	if len(overrides) > 0 && overrides[0] != nil {
		return overrides[0]
	}
	return DefaultWiFiRuntimeState()
}

func ConfigEventChannel(state ...*wifiRuntimeState) chan string {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.Lock()
	defer resolvedState.wifiStateMu.Unlock()
	if resolvedState.confChannel == nil {
		resolvedState.confChannel = make(chan string, 100)
	}
	return resolvedState.confChannel
}

// ConfChannel is a backwards-compatible alias for ConfigEventChannel.
func ConfChannel(state ...*wifiRuntimeState) chan string {
	return ConfigEventChannel(state...)
}

func LocalUrl(state ...*wifiRuntimeState) string {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.RLock()
	defer resolvedState.wifiStateMu.RUnlock()
	return resolvedState.localUrl
}

func SetLocalUrl(localURL string, state ...*wifiRuntimeState) {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.Lock()
	defer resolvedState.wifiStateMu.Unlock()
	resolvedState.localUrl = localURL
}

func WiFiDevice(state ...*wifiRuntimeState) string {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.RLock()
	defer resolvedState.wifiStateMu.RUnlock()
	return resolvedState.device
}

func setWiFiDevice(device string, state ...*wifiRuntimeState) {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.Lock()
	defer resolvedState.wifiStateMu.Unlock()
	resolvedState.device = device
}

func setWifiInfo(info structs.SystemWifi, state ...*wifiRuntimeState) {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.Lock()
	defer resolvedState.wifiStateMu.Unlock()
	resolvedState.wifiInfo = info
}

func WifiInfoSnapshot(state ...*wifiRuntimeState) structs.SystemWifi {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.wifiStateMu.RLock()
	defer resolvedState.wifiStateMu.RUnlock()
	return resolvedState.wifiInfo
}

func HasWifiDevice() bool {
	return HasWifiDeviceWithState()
}

func HasWifiDeviceWithState(state ...*wifiRuntimeState) bool {
	return WiFiDevice(state...) != ""
}

func InitializeWiFi() error {
	return InitializeWiFiWithState(DefaultWiFiRuntimeState())
}

func InitializeWiFiWithState(state ...*wifiRuntimeState) error {
	resolvedState := resolveWiFiRuntimeState(state...)
	var initErr error
	resolvedState.wifiInit.Do(func() {
		initErr = NewWiFiRuntimeService().StartWiFiInfoLoop(context.Background())
	})
	return initErr
}

func IsC2CMode(state ...*wifiRuntimeState) bool {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.c2cMu.Lock()
	defer resolvedState.c2cMu.Unlock()
	return resolvedState.c2cEnabled
}

func SetC2CMode(isTrue bool) error {
	return SetC2CModeWithState(isTrue)
}

func SetC2CModeWithState(isTrue bool, state ...*wifiRuntimeState) error {
	resolvedState := resolveWiFiRuntimeState(state...)
	resolvedState.c2cMu.Lock()
	defer resolvedState.c2cMu.Unlock()
	resolvedState.c2cEnabled = isTrue
	return nil
}
