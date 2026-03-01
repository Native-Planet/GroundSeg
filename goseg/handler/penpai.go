package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"runtime"

	"go.uber.org/zap"
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
	zap.L().Info("Penpai")
	var penpaiPayload structs.WsPenpaiPayload
	err := json.Unmarshal(msg, &penpaiPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal penpai payload: %v", err)
	}
	conf := confForPenpai()
	switch penpaiPayload.Payload.Action {
	case "toggle":
		running := false
		if conf.PenpaiRunning {
			// stop container
			err := stopContainerByNameForPenpai("llama-gpt-api")
			if err != nil {
				return fmt.Errorf("Failed to stop Llama API: %v", err)
			}
			err = stopContainerByNameForPenpai("llama-gpt-ui")
			if err != nil {
				return fmt.Errorf("Failed to stop Llama UI: %v", err)
			}
		} else {
			// start container
			info, err := startContainerForPenpai("llama-gpt-api", "llama-api")
			if err != nil {
				return fmt.Errorf("Error starting Llama API: %v", err)
			}
			updateContainerStateForPenpai("llama-api", info)
			running = true
		}
		if err = updateConfTypedForPenpai(withPenpaiRunningForPenpai(running)); err != nil {
			return fmt.Errorf("%v", err)
		}
		return nil
	case "set-model":
		// update config
		model := penpaiPayload.Payload.Model
		if err = updateConfTypedForPenpai(withPenpaiActiveForPenpai(model)); err != nil {
			return fmt.Errorf("%v", err)
		}
		if err := deleteContainerForPenpai("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		// if running, restart container
		if conf.PenpaiRunning {
			if _, err := startContainerForPenpai("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
	case "set-cores":
		cores := penpaiPayload.Payload.Cores
		// check if core count is valid
		if cores < 1 {
			return fmt.Errorf("Penpai unable to set 0 cores!")
		}
		if cores >= numCPUForPenpai() {
			return fmt.Errorf("Penpai unable to set %v cores!", cores)
		}
		// update config
		if err = updateConfTypedForPenpai(withPenpaiCoresForPenpai(cores)); err != nil {
			return fmt.Errorf("%v", err)
		}
		if err := deleteContainerForPenpai("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		// if running, restart container
		if conf.PenpaiRunning {
			if _, err := startContainerForPenpai("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
		return nil
	case "remove":
		// check if container exists
		// remove container, delete volume
		zap.L().Debug(fmt.Sprintf("Todo: remove penpai"))
	}
	return nil
}
