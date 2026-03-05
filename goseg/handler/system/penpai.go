package system

import (
	"encoding/json"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/handler/systemsvc"
	"runtime"
)

var (
	confForPenpai                 = config.Config
	stopContainerByNameForPenpai  = orchestration.StopContainerByName
	startContainerForPenpai       = orchestration.StartContainer
	updateContainerStateForPenpai = config.UpdateContainerState
	updateConfTypedForPenpai      = config.UpdateConfigTyped
	withPenpaiRunningForPenpai    = config.WithPenpaiRunning
	withPenpaiActiveForPenpai     = config.WithPenpaiActive
	withPenpaiCoresForPenpai      = config.WithPenpaiCores
	deleteContainerForPenpai      = orchestration.DeleteContainer
	numCPUForPenpai               = runtime.NumCPU
)

func PenpaiHandler(msg []byte) error {
	return systemsvc.HandlePenpai(msg, systemsvc.PenpaiDependencies{
		Unmarshal:            json.Unmarshal,
		Config:               confForPenpai,
		StopContainerByName:  stopContainerByNameForPenpai,
		StartContainerByName: startContainerForPenpai,
		UpdateContainerState: updateContainerStateForPenpai,
		UpdateConfTyped:      updateConfTypedForPenpai,
		WithPenpaiRunning:    withPenpaiRunningForPenpai,
		WithPenpaiActive:     withPenpaiActiveForPenpai,
		WithPenpaiCores:      withPenpaiCoresForPenpai,
		DeleteContainer:      deleteContainerForPenpai,
		NumCPU:               numCPUForPenpai,
	})
}
