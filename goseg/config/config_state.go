package config

import (
	"encoding/json"
	"fmt"

	"groundseg/config/containerstate"
	"groundseg/config/runtimeutil"
	"groundseg/structs"

	"go.uber.org/zap"
)

// UpdateContainerState updates desired/actual state for a named container.
func UpdateContainerState(name string, containerState structs.ContainerState) {
	containerstate.Update(name, containerState)
	logMsg := "<hidden>"
	res, _ := json.Marshal(containerState)
	zap.L().Info(fmt.Sprintf("%s state:%s", name, logMsg))
	zap.L().Debug(fmt.Sprintf("%s state:%s", name, string(res)))
}

// DeleteContainerState removes a container from runtime container state tracking.
func DeleteContainerState(name string) {
	containerstate.Delete(name)
	zap.L().Debug(fmt.Sprintf("%s removed from container state map", name))
}

// GetContainerState returns a defensive copy of runtime container state.
func GetContainerState() map[string]structs.ContainerState {
	return containerstate.Snapshot()
}

// NetCheck checks outbound tcp connectivity for an ip:port endpoint.
func NetCheck(netCheck string) bool {
	internet := runtimeutil.NetCheck(netCheck)
	if !internet {
		zap.L().Error(fmt.Sprintf("Check internet access error: unable to reach %s", netCheck))
	}
	return internet
}

// RandString generates a random secret string of the input length.
func RandString(length int) string {
	randomValue, err := RandStringWithError(length)
	if err != nil {
		zap.L().Warn("Random error :s", zap.Error(err))
		return ""
	}
	return randomValue
}

// RandStringWithError generates a random secret string of the input length.
func RandStringWithError(length int) (string, error) {
	return runtimeutil.RandStringWithError(length)
}

// GetSHA256 returns the hex-encoded SHA256 for a file.
func GetSHA256(filePath string) (string, error) {
	return runtimeutil.SHA256(filePath)
}
