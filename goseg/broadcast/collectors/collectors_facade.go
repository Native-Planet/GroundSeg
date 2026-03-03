package collectors

import (
	"groundseg/structs"
	"time"
)

// BroadcastPierCollectorContract handles only ship/urbit pier state assembly.
type BroadcastPierCollectorContract interface {
	CollectPierInfo(existing map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error)
}

// BroadcastInfoCollectorContract handles non-StarTram metadata collections.
type BroadcastInfoCollectorContract interface {
	CollectAppsInfo() structs.Apps
	CollectProfileInfo(regions map[string]structs.StartramRegion) structs.Profile
	CollectSystemInfo() structs.System
}

// BroadcastStartramCollectorContract handles Startram-specific state collection.
type BroadcastStartramCollectorContract interface {
	LoadStartramRegions() (map[string]structs.StartramRegion, error)
}

// BroadcastCollectorContract is the historical broad contract and embeds focused collectors.
type BroadcastCollectorContract interface {
	BroadcastPierCollectorContract
	BroadcastInfoCollectorContract
	BroadcastStartramCollectorContract
}

type broadcastPierCollectorContract struct{}
type broadcastInfoCollectorContract struct{}
type broadcastStartramCollectorContract struct{}

type broadcastCollectorContract struct {
	BroadcastPierCollectorContract
	BroadcastInfoCollectorContract
	BroadcastStartramCollectorContract
}

var (
	defaultPierCollectorContract      BroadcastPierCollectorContract     = broadcastPierCollectorContract{}
	defaultInfoCollectorContract      BroadcastInfoCollectorContract     = broadcastInfoCollectorContract{}
	defaultStartramCollectorContract  BroadcastStartramCollectorContract = broadcastStartramCollectorContract{}
	defaultBroadcastCollectorContract BroadcastCollectorContract         = broadcastCollectorContract{
		BroadcastPierCollectorContract:     broadcastPierCollectorContract{},
		BroadcastInfoCollectorContract:     broadcastInfoCollectorContract{},
		BroadcastStartramCollectorContract: broadcastStartramCollectorContract{},
	}
)

func DefaultBroadcastPierCollectorContract() BroadcastPierCollectorContract {
	return defaultPierCollectorContract
}

func DefaultBroadcastInfoCollectorContract() BroadcastInfoCollectorContract {
	return defaultInfoCollectorContract
}

func DefaultBroadcastStartramCollectorContract() BroadcastStartramCollectorContract {
	return defaultStartramCollectorContract
}

func DefaultBroadcastCollectorContract() BroadcastCollectorContract {
	return defaultBroadcastCollectorContract
}

func SetBroadcastPierCollectorContract(contract BroadcastPierCollectorContract) {
	if contract != nil {
		defaultPierCollectorContract = contract
	}
}

func SetBroadcastInfoCollectorContract(contract BroadcastInfoCollectorContract) {
	if contract != nil {
		defaultInfoCollectorContract = contract
	}
}

func SetBroadcastStartramCollectorContract(contract BroadcastStartramCollectorContract) {
	if contract != nil {
		defaultStartramCollectorContract = contract
	}
}

func SetBroadcastCollectorContract(contract BroadcastCollectorContract) {
	if contract == nil {
		return
	}
	SetBroadcastPierCollectorContract(contract)
	SetBroadcastInfoCollectorContract(contract)
	SetBroadcastStartramCollectorContract(contract)
	defaultBroadcastCollectorContract = contract
}

func (broadcastPierCollectorContract) CollectPierInfo(existing map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return ConstructPierInfo(existing, scheduled)
}

func (broadcastInfoCollectorContract) CollectAppsInfo() structs.Apps {
	return ConstructAppsInfo()
}

func (broadcastInfoCollectorContract) CollectProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	return ConstructProfileInfo(regions)
}

func (broadcastInfoCollectorContract) CollectSystemInfo() structs.System {
	return ConstructSystemInfo()
}

func (broadcastStartramCollectorContract) LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	return LoadStartramRegions()
}
