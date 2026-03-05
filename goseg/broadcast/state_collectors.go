package broadcast

import (
	"groundseg/broadcast/collectors"
	"groundseg/structs"
	"time"
)

func (runtime *broadcastStateRuntime) pierCollectorContract() collectors.BroadcastPierCollectorContract {
	if runtime != nil && runtime.pierCollector != nil {
		return runtime.pierCollector
	}
	return collectors.DefaultBroadcastPierCollectorContract()
}

func (runtime *broadcastStateRuntime) infoCollectorContract() collectors.BroadcastInfoCollectorContract {
	if runtime != nil && runtime.infoCollector != nil {
		return runtime.infoCollector
	}
	return collectors.DefaultBroadcastInfoCollectorContract()
}

func (runtime *broadcastStateRuntime) startramCollectorContract() collectors.BroadcastStartramCollectorContract {
	if runtime != nil && runtime.startramCollector != nil {
		return runtime.startramCollector
	}
	return collectors.DefaultBroadcastStartramCollectorContract()
}

func (runtime *broadcastStateRuntime) collectPierInfo(urbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return runtime.pierCollectorContract().CollectPierInfo(urbits, scheduled)
}

func (runtime *broadcastStateRuntime) collectAppsInfo() structs.Apps {
	return runtime.infoCollectorContract().CollectAppsInfo()
}

func (runtime *broadcastStateRuntime) collectProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	return runtime.infoCollectorContract().CollectProfileInfo(regions)
}

func (runtime *broadcastStateRuntime) collectSystemInfo() structs.System {
	return runtime.infoCollectorContract().CollectSystemInfo()
}
