package shipworkflow

import "groundseg/config"

var (
	getUrbitConfigFn            = config.UrbitConf
	loadUrbitConfigFn           = config.LoadUrbitConfig
	getContainerStatesFn        = config.GetContainerState
	updateContainerStateFn      = config.UpdateContainerState
	getStartramSettingsSnapshot = config.StartramSettingsSnapshot
)
