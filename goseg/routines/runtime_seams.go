package routines

import (
	"context"
	"groundseg/chopsvc"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/handler/ship"
	"groundseg/structs"
	"net/http"
	"time"
)

func newVersionRuntime() versionRuntime {
	return versionRuntime{
		channelOps: versionChannelOps{
			syncVersionInfoFn:   config.SyncVersionInfo,
			getVersionChannelFn: config.GetVersionChannel,
			setVersionChannelFn: config.SetVersionChannel,
		},
		configOps: versionConfigOps{
			getConfFn:   config.Conf,
			getSha256Fn: getSha256,
			architectureFn: func() string {
				return config.RuntimeContextSnapshot().Architecture
			},
			basePathFn: func() string {
				return config.RuntimeContextSnapshot().BasePath
			},
			debugModeFn: func() bool {
				return config.RuntimeContextSnapshot().DebugMode
			},
			getShipStatusFn:            orchestration.GetShipStatus,
			startContainerFn:           orchestration.StartContainer,
			stopContainerFn:            orchestration.StopContainerByName,
			loadUrbitConfigFn:          config.LoadUrbitConfig,
			urbitConfFn:                config.UrbitConf,
			updateUrbitRuntimeConfigFn: config.UpdateUrbitRuntimeConfig,
			waitCompleteFn:             ship.WaitComplete,
			chopPierFn:                 chopsvc.ChopPier,
			updateUrbitFn:              config.UpdateUrbit,
		},
		updateOps: versionUpdateOps{
			updateDockerFn: updateDockerForRuntime,
			updateBinaryFn: updateBinary,
			downloadFn: func(ctx context.Context, url string) (*http.Response, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					return nil, err
				}
				return versionUpdateHTTPClient.Do(req)
			},
		},
	}
}

// version runtime abstractions for periodic updates.
type versionChannelOps struct {
	syncVersionInfoFn   func() (structs.Channel, bool)
	getVersionChannelFn func() structs.Channel
	setVersionChannelFn func(structs.Channel)
}

type versionConfigOps struct {
	getConfFn                  func() structs.SysConfig
	getSha256Fn                func(string) (string, error)
	architectureFn             func() string
	basePathFn                 func() string
	debugModeFn                func() bool
	getShipStatusFn            func([]string) (map[string]string, error)
	startContainerFn           func(string, string) (structs.ContainerState, error)
	stopContainerFn            func(string) error
	loadUrbitConfigFn          func(string) error
	urbitConfFn                func(string) structs.UrbitDocker
	updateUrbitRuntimeConfigFn func(string, func(*structs.UrbitRuntimeConfig) error) error
	waitCompleteFn             func(string) error
	chopPierFn                 func(string) error
	updateUrbitFn              func(string, func(*structs.UrbitDocker) error) error
}

type versionUpdateOps struct {
	updateDockerFn func(versionConfigOps, string, structs.Channel, structs.Channel) error
	updateBinaryFn func(context.Context, versionUpdateOps, versionConfigOps, string, structs.Channel) error
	downloadFn     func(context.Context, string) (*http.Response, error)
}

var versionUpdateHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

type versionRuntime struct {
	channelOps versionChannelOps
	configOps  versionConfigOps
	updateOps  versionUpdateOps
}
