package collectors

import (
	"fmt"
	"go.uber.org/zap"
	"groundseg/structs"
	"groundseg/system"
	"runtime"
	"strings"
)

func ConstructAppsInfo() structs.Apps {
	runtime := DefaultCollectorRuntime()
	apps, err := appInfoCollector{}.collect(runtime)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("constructing apps info failed: %v", err))
		return structs.Apps{}
	}
	return apps
}

type appInfoCollector struct{}

func (appInfoCollector) collect(runtimeRt collectorRuntime) (structs.Apps, error) {
	var apps structs.Apps
	settings := runtimeRt.PenpaiSettingsFn()
	var modelTitles []string
	for _, penpaiInfo := range settings.Models {
		modelTitles = append(modelTitles, penpaiInfo.ModelTitle)
	}
	apps.Penpai.Info.Models = modelTitles
	apps.Penpai.Info.Allowed = settings.Allowed
	apps.Penpai.Info.ActiveModel = settings.ActiveModel
	apps.Penpai.Info.Running = settings.Running
	apps.Penpai.Info.MaxCores = runtime.NumCPU() - 1
	apps.Penpai.Info.ActiveCores = settings.ActiveCores
	return apps, nil
}

func ConstructProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	runtime := DefaultCollectorRuntime()
	profile, err := profileInfoCollector{}.collect(runtime, regions)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("constructing profile info failed: %v", err))
		return structs.Profile{}
	}
	return profile
}

type profileInfoCollector struct{}

func (profileInfoCollector) collect(runtimeRt collectorRuntime, regions map[string]structs.StartramRegion) (structs.Profile, error) {
	var startramInfo structs.Startram
	settings := runtimeRt.StartramSettingsFn()
	startramConfig := runtimeRt.StartramConfigFn()
	backupTime := runtimeRt.BackupTimeFn()
	startramInfo.Info.Registered = settings.WgRegistered
	startramInfo.Info.Running = settings.WgOn
	startramInfo.Info.Endpoint = settings.EndpointURL
	startramInfo.Info.RemoteBackupReady = settings.RemoteBackupPassword != ""
	startramInfo.Info.BackupTime = backupTime.Format("3:04PM MST")

	startramInfo.Info.Region = startramConfig.Region
	startramInfo.Info.Expiry = startramConfig.Lease
	startramInfo.Info.Renew = startramConfig.Ongoing == 0
	startramInfo.Info.UrlID = startramConfig.UrlID

	startramServices := []string{}
	for _, subdomain := range startramConfig.Subdomains {
		parts := strings.Split(subdomain.URL, ".")
		if len(parts) < 3 {
			zap.L().Warn(fmt.Sprintf("startram services information invalid url: %s", subdomain.URL))
			continue
		}
		patp := parts[len(parts)-3]
		shipExists := false
		for _, ship := range startramServices {
			if ship == patp {
				shipExists = true
				break
			}
		}
		if !shipExists {
			startramServices = append(startramServices, patp)
		}
	}
	startramInfo.Info.StartramServices = startramServices
	startramInfo.Info.Regions = regions

	var profile structs.Profile
	profile.Startram = startramInfo
	return profile, nil
}

func ConstructSystemInfo() structs.System {
	runtime := DefaultCollectorRuntime()
	systemInfo, err := systemInfoCollector{}.collect(runtime)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("constructing system info failed: %v", err))
		return structs.System{}
	}
	return systemInfo
}

type systemInfoCollector struct{}

func (systemInfoCollector) collect(runtimeRt collectorRuntime) (structs.System, error) {
	swapSettings := runtimeRt.SwapSettingsFn()
	return system.CollectBroadcastSystemInfo(swapSettings.SwapVal)
}
