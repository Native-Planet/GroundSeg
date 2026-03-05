package collectors

import (
	"testing"
	"time"

	"groundseg/structs"
)

type stubPierCollector struct{}

func (stubPierCollector) CollectPierInfo(existing map[string]structs.Urbit, _ func(string) time.Time) (map[string]structs.Urbit, error) {
	return existing, nil
}

type stubInfoCollector struct{}

func (stubInfoCollector) CollectAppsInfo() structs.Apps { return structs.Apps{} }
func (stubInfoCollector) CollectProfileInfo(_ map[string]structs.StartramRegion) structs.Profile {
	return structs.Profile{}
}
func (stubInfoCollector) CollectSystemInfo() structs.System { return structs.System{} }

type stubStartramCollector struct{}

func (stubStartramCollector) LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	return map[string]structs.StartramRegion{}, nil
}

type stubCollector struct {
	BroadcastPierCollectorContract
	BroadcastInfoCollectorContract
	BroadcastStartramCollectorContract
}

func TestSetBroadcastCollectorContractsOverridesDefaults(t *testing.T) {
	origPier := defaultPierCollectorContract
	origInfo := defaultInfoCollectorContract
	origStartram := defaultStartramCollectorContract
	origCombined := defaultBroadcastCollectorContract
	t.Cleanup(func() {
		defaultPierCollectorContract = origPier
		defaultInfoCollectorContract = origInfo
		defaultStartramCollectorContract = origStartram
		defaultBroadcastCollectorContract = origCombined
	})

	pier := stubPierCollector{}
	info := stubInfoCollector{}
	startram := stubStartramCollector{}
	combined := stubCollector{
		BroadcastPierCollectorContract:     pier,
		BroadcastInfoCollectorContract:     info,
		BroadcastStartramCollectorContract: startram,
	}

	SetBroadcastCollectorContract(combined)

	if got := DefaultBroadcastPierCollectorContract(); got != combined {
		t.Fatalf("expected overridden pier collector contract")
	}
	if got := DefaultBroadcastInfoCollectorContract(); got != combined {
		t.Fatalf("expected overridden info collector contract")
	}
	if got := DefaultBroadcastStartramCollectorContract(); got != combined {
		t.Fatalf("expected overridden startram collector contract")
	}
	if got := DefaultBroadcastCollectorContract(); got != combined {
		t.Fatalf("expected overridden combined collector contract")
	}
}
