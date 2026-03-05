package broadcast

import (
	"errors"
	"testing"
	"time"

	"groundseg/structs"
)

type testPierCollector struct {
	urbits map[string]structs.Urbit
	err    error
}

func (collector testPierCollector) CollectPierInfo(map[string]structs.Urbit, func(string) time.Time) (map[string]structs.Urbit, error) {
	if collector.err != nil {
		return nil, collector.err
	}
	return collector.urbits, nil
}

type testInfoCollector struct {
	apps    structs.Apps
	profile structs.Profile
	system  structs.System
}

func (collector testInfoCollector) CollectAppsInfo() structs.Apps {
	return collector.apps
}

func (collector testInfoCollector) CollectProfileInfo(map[string]structs.StartramRegion) structs.Profile {
	return collector.profile
}

func (collector testInfoCollector) CollectSystemInfo() structs.System {
	return collector.system
}

type testStartramCollector struct {
	regions map[string]structs.StartramRegion
	err     error
}

func (collector testStartramCollector) LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	if collector.err != nil {
		return nil, collector.err
	}
	return collector.regions, nil
}

func TestBootstrapBroadcastStateRequiresRuntime(t *testing.T) {
	if err := bootstrapBroadcastState(nil); err == nil {
		t.Fatal("expected bootstrapBroadcastState to fail with nil runtime")
	} else if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}

func TestLoadStartramRegionsWithRuntimeRequiresRuntime(t *testing.T) {
	if err := LoadStartramRegionsWithRuntime(nil); err == nil {
		t.Fatal("expected LoadStartramRegionsWithRuntime to fail with nil runtime")
	} else if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}

func TestReloadUrbitsWithRuntimeRequiresRuntime(t *testing.T) {
	if err := ReloadUrbitsWithRuntime(nil); err == nil {
		t.Fatal("expected ReloadUrbitsWithRuntime to fail with nil runtime")
	} else if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}

func TestLoadStartramRegionsWithRuntimeUpdatesRuntimeState(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	runtime.startramCollector = testStartramCollector{
		regions: map[string]structs.StartramRegion{
			"us-east": {Country: "US", Desc: "East"},
		},
	}

	if err := LoadStartramRegionsWithRuntime(runtime); err != nil {
		t.Fatalf("LoadStartramRegionsWithRuntime returned error: %v", err)
	}

	gotRegions := runtime.GetState().Profile.Startram.Info.Regions
	if len(gotRegions) != 1 || gotRegions["us-east"].Country != "US" {
		t.Fatalf("unexpected regions in broadcast state: %#v", gotRegions)
	}
}

func TestLoadStartramRegionsWithRuntimeReturnsCollectorError(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	runtime.startramCollector = testStartramCollector{err: errors.New("collector unavailable")}

	if err := LoadStartramRegionsWithRuntime(runtime); err == nil {
		t.Fatal("expected collector error to be returned")
	}
}

func TestReloadUrbitsWithRuntimeUsesPierCollector(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	runtime.pierCollector = testPierCollector{
		urbits: map[string]structs.Urbit{
			"zod": {},
		},
	}

	if err := ReloadUrbitsWithRuntime(runtime); err != nil {
		t.Fatalf("ReloadUrbitsWithRuntime returned error: %v", err)
	}

	if _, ok := runtime.GetState().Urbits["zod"]; !ok {
		t.Fatalf("expected reloaded urbit to be present in runtime state")
	}
}

func TestCollectorContractsPreferRuntimeOverrides(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	pierCollector := testPierCollector{}
	infoCollector := testInfoCollector{}
	startramCollector := testStartramCollector{}
	runtime.pierCollector = pierCollector
	runtime.infoCollector = infoCollector
	runtime.startramCollector = startramCollector

	if _, ok := runtime.pierCollectorContract().(testPierCollector); !ok {
		t.Fatal("expected runtime pier collector override")
	}
	if _, ok := runtime.infoCollectorContract().(testInfoCollector); !ok {
		t.Fatal("expected runtime info collector override")
	}
	if _, ok := runtime.startramCollectorContract().(testStartramCollector); !ok {
		t.Fatal("expected runtime startram collector override")
	}
}
