package broadcast

import (
	"errors"
	"fmt"

	"groundseg/structs"

	"go.uber.org/zap"
)

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("bootstrap broadcast state: %w", errBroadcastRuntimeRequired)
	}
	return bootstrapBroadcastStateWithRuntime(runtime)
}

func bootstrapBroadcastStateWithRuntime(resolved *broadcastStateRuntime) error {
	if resolved == nil {
		return fmt.Errorf("bootstrap broadcast state: %w", errBroadcastRuntimeRequired)
	}
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	state := resolved.GetState()
	urbits, err := resolved.collectPierInfo(state.Urbits, resolved.GetScheduledPack)
	if err != nil {
		return fmt.Errorf("bootstrap broadcast state: %w", err)
	}
	nextState := structs.AuthBroadcast{
		Urbits:  urbits,
		System:  resolved.collectSystemInfo(),
		Profile: resolved.collectProfileInfo(state.Profile.Startram.Info.Regions),
		Apps:    resolved.collectAppsInfo(),
	}
	resolved.Lock()
	resolved.broadcastState = cloneBroadcastState(nextState)
	resolved.Unlock()
	// start looping info refreshes
	if err := StartBroadcastLoopWithRuntime(newBroadcastLoopRuntime(resolved)); err != nil {
		return fmt.Errorf("broadcast loop failed to start: %w", err)
	}
	return nil
}

func LoadStartramRegionsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return fmt.Errorf("load startram regions: %w", errBroadcastRuntimeRequired)
	}
	return LoadStartramRegionsWithRuntimeState(runtime)
}

func LoadStartramRegionsWithRuntimeState(resolved *broadcastStateRuntime) error {
	if resolved == nil {
		return fmt.Errorf("load startram regions: %w", errBroadcastRuntimeRequired)
	}
	zap.L().Info("Retrieving StarTram region info")
	regions, err := resolved.startramCollectorContract().LoadStartramRegions()
	if err != nil {
		return fmt.Errorf("load startram regions: %w", err)
	}
	resolved.Lock()
	resolved.broadcastState.Profile.Startram.Info.Regions = regions
	resolved.Unlock()
	return nil
}

func ReloadUrbits() error {
	return ReloadUrbitsWithRuntime(DefaultBroadcastStateRuntime())
}

func ReloadUrbitsWithRuntime(runtime *broadcastStateRuntime) error {
	if runtime == nil {
		return errors.New("broadcast state runtime is required to reload urbits")
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
