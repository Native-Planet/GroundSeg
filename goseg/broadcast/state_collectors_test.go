package broadcast

import (
	"testing"
	"time"

	"groundseg/structs"
)

type stubPierCollector struct {
	collect func(map[string]structs.Urbit, func(string) time.Time) (map[string]structs.Urbit, error)
}

func (collector stubPierCollector) CollectPierInfo(existing map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return collector.collect(existing, scheduled)
}

type stubInfoCollector struct {
	apps    structs.Apps
	profile structs.Profile
	system  structs.System
}

func (collector stubInfoCollector) CollectAppsInfo() structs.Apps {
	return collector.apps
}

func (collector stubInfoCollector) CollectProfileInfo(map[string]structs.StartramRegion) structs.Profile {
	return collector.profile
}

func (collector stubInfoCollector) CollectSystemInfo() structs.System {
	return collector.system
}

type stubStartramCollector struct {
	regions map[string]structs.StartramRegion
}

func (collector stubStartramCollector) LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	return collector.regions, nil
}

func TestBroadcastCollectorsUseInjectedContracts(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	expectedPack := time.Now().Add(time.Minute)
	runtime.pierCollector = stubPierCollector{
		collect: func(existing map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
			urbit := structs.Urbit{}
			urbit.Info.NextPack = scheduled("~nec").Format(time.RFC3339)
			return map[string]structs.Urbit{
				"~nec": urbit,
			}, nil
		},
	}
	apps := structs.Apps{}
	apps.Penpai.Info.ActiveModel = "apps"
	profile := structs.Profile{}
	profile.Startram.Info.Endpoint = "profile"
	systemInfo := structs.System{}
	systemInfo.Transition.Type = "system"
	runtime.infoCollector = stubInfoCollector{
		apps:    apps,
		profile: profile,
		system:  systemInfo,
	}
	runtime.startramCollector = stubStartramCollector{
		regions: map[string]structs.StartramRegion{"us-east": {}},
	}
	_ = runtime.UpdateScheduledPack("~nec", expectedPack)

	urbits, err := runtime.collectPierInfo(nil, runtime.GetScheduledPack)
	if err != nil {
		t.Fatalf("collectPierInfo returned error: %v", err)
	}
	if urbits["~nec"].Info.NextPack != expectedPack.Format(time.RFC3339) {
		t.Fatalf("expected injected scheduled pack in pier info, got %q", urbits["~nec"].Info.NextPack)
	}
	if runtime.collectAppsInfo().Penpai.Info.ActiveModel != "apps" {
		t.Fatal("expected injected apps collector")
	}
	if runtime.collectProfileInfo(nil).Startram.Info.Endpoint != "profile" {
		t.Fatal("expected injected profile collector")
	}
	if runtime.collectSystemInfo().Transition.Type != "system" {
		t.Fatal("expected injected system collector")
	}
	regions, err := runtime.startramCollectorContract().LoadStartramRegions()
	if err != nil {
		t.Fatalf("LoadStartramRegions returned error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected injected startram regions, got %#v", regions)
	}
}
