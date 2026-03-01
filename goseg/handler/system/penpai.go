package system

import (
	"encoding/json"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/handler/systemsvc"
	"runtime"
)

var (
	confForPenpai                 = config.Conf
	stopContainerByNameForPenpai  = docker.StopContainerByName
	startContainerForPenpai       = docker.StartContainer
	updateContainerStateForPenpai = config.UpdateContainerState
	updateConfTypedForPenpai      = config.UpdateConfTyped
	withPenpaiRunningForPenpai    = config.WithPenpaiRunning
	withPenpaiActiveForPenpai     = config.WithPenpaiActive
	withPenpaiCoresForPenpai      = config.WithPenpaiCores
	deleteContainerForPenpai      = docker.DeleteContainer
	numCPUForPenpai               = runtime.NumCPU
)

func PenpaiHandler(msg []byte) error {
	return systemsvc.HandlePenpai(msg, systemsvc.PenpaiDependencies{
		Unmarshal:            json.Unmarshal,
		Conf:                 confForPenpai,
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
