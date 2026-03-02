package orchestration

import (
	"groundseg/docker/orchestration/container"

	dockerc "github.com/docker/docker/api/types/container"
)

func LoadNetdata() error {
	return loadNetdataWithRuntime(netdataRuntimeFromDocker(newDockerRuntime()))
}

func loadNetdataWithRuntime(rt container.NetdataRuntime) error {
	return container.LoadNetdataWithRuntime(rt)
}

func netdataContainerConf() (dockerc.Config, dockerc.HostConfig, error) {
	return netdataContainerConfWithRuntime(netdataRuntimeFromDocker(newDockerRuntime()))
}

func netdataContainerConfWithRuntime(rt container.NetdataRuntime) (dockerc.Config, dockerc.HostConfig, error) {
	return container.NetdataContainerConfWithRuntime(rt)
}

func WriteNDConf() error {
	return writeNDConfWithRuntime(netdataRuntimeFromDocker(newDockerRuntime()))
}

func writeNDConfWithRuntime(rt container.NetdataRuntime) error {
	return container.WriteNDConfWithRuntime(rt)
}

func writeNDConfToFile(filePath string, content string) error {
	return writeNDConfToFileWithRuntime(netdataRuntimeFromDocker(newDockerRuntime()), filePath, content)
}

func writeNDConfToFileWithRuntime(rt container.NetdataRuntime, filePath string, content string) error {
	return container.WriteNDConfToFileWithRuntime(rt, filePath, content)
}

func copyNDFileToVolume(filePath string, targetPath string, volumeName string) error {
	return copyNDFileToVolumeWithRuntime(netdataRuntimeFromDocker(newDockerRuntime()), filePath, targetPath, volumeName)
}

func copyNDFileToVolumeWithRuntime(rt container.NetdataRuntime, filePath string, targetPath string, volumeName string) error {
	return container.CopyNDFileToVolumeWithRuntime(rt, filePath, targetPath, volumeName)
}
