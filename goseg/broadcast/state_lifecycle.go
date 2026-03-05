package broadcast

import (
	"fmt"

	"groundseg/structs"

	"go.uber.org/zap"
)

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("bootstrap broadcast state: %w", ErrBroadcastRuntimeRequired)
	}
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	state := runtime.GetState()
	urbits, err := runtime.collectPierInfo(state.Urbits, runtime.GetScheduledPack)
	if err != nil {
		return fmt.Errorf("bootstrap broadcast state: %w", err)
	}
	nextState := structs.AuthBroadcast{
		Urbits:  urbits,
		System:  runtime.collectSystemInfo(),
		Profile: runtime.collectProfileInfo(state.Profile.Startram.Info.Regions),
		Apps:    runtime.collectAppsInfo(),
	}
	runtime.Lock()
	runtime.broadcastState = cloneBroadcastState(nextState)
	runtime.Unlock()
	// start looping info refreshes
	if err := StartBroadcastLoopWithRuntime(newBroadcastLoopRuntime(runtime)); err != nil {
		return fmt.Errorf("broadcast loop failed to start: %w", err)
	}
	return nil
}

func LoadStartramRegionsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("load startram regions: %w", ErrBroadcastRuntimeRequired)
	}
	zap.L().Info("Retrieving StarTram region info")
	regions, err := runtime.startramCollectorContract().LoadStartramRegions()
	if err != nil {
		return fmt.Errorf("load startram regions: %w", err)
	}
	runtime.Lock()
	runtime.broadcastState.Profile.Startram.Info.Regions = regions
	runtime.Unlock()
	return nil
}

func ReloadUrbits() error {
	return ReloadUrbitsWithRuntime(DefaultBroadcastStateRuntime())
}

func ReloadUrbitsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("reload urbits: %w", ErrBroadcastRuntimeRequired)
	}
	zap.L().Info("Reloading ships in broadcast")
	urbits, err := runtime.collectPierInfo(runtime.GetState().Urbits, runtime.GetScheduledPack)
	if err != nil {
		return fmt.Errorf("reload urbit states for broadcast: %w", err)
	}
	runtime.Lock()
	runtime.broadcastState.Urbits = urbits
	runtime.Unlock()
	return nil
}
