package orchestration

import (
	"groundseg/docker/orchestration/container"
	"groundseg/structs"

	dockerc "github.com/docker/docker/api/types/container"
)

func LoadMC() error {
	return loadMCWithRuntime(minioRuntimeFromDocker(newDockerRuntime()))
}

func loadMCWithRuntime(rt container.MinioRuntime) error {
	return container.LoadMCWithRuntime(rt)
}

func LoadMinIOs() error {
	return loadMinIOsWithRuntime(minioRuntimeFromDocker(newDockerRuntime()))
}

func loadMinIOsWithRuntime(rt container.MinioRuntime) error {
	return container.LoadMinIOsWithRuntime(rt)
}

func minioContainerConf(containerName string) (dockerc.Config, dockerc.HostConfig, error) {
	return minioContainerConfWithRuntime(minioRuntimeFromDocker(newDockerRuntime()), containerName)
}

func minioContainerConfWithRuntime(rt container.MinioRuntime, containerName string) (dockerc.Config, dockerc.HostConfig, error) {
	return container.MinioContainerConfWithRuntime(rt, containerName)
}

func mcContainerConf() (dockerc.Config, dockerc.HostConfig, error) {
	return mcContainerConfWithRuntime(minioRuntimeFromDocker(newDockerRuntime()))
}

func mcContainerConfWithRuntime(rt container.MinioRuntime) (dockerc.Config, dockerc.HostConfig, error) {
	return container.MCContainerConfWithRuntime(rt)
}

func setMinIOAdminAccount(containerName string) error {
	return setMinIOAdminAccountWithRuntime(minioRuntimeFromDocker(newDockerRuntime()), containerName)
}

func setMinIOAdminAccountWithRuntime(rt container.MinioRuntime, containerName string) error {
	return container.SetMinIOAdminAccountWithRuntime(rt, containerName)
}

func getPatpFromMinIOName(containerName string) (string, error) {
	return container.GetPatpFromMinIOName(containerName)
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	return createMinIOServiceAccountWithRuntime(minioRuntimeFromDocker(newDockerRuntime()), patp)
}

func createMinIOServiceAccountWithRuntime(rt container.MinioRuntime, patp string) (structs.MinIOServiceAccount, error) {
	return container.CreateMinIOServiceAccountWithRuntime(rt, patp)
}

func isNonFatalMinIOServiceAccountErr(exitCode int, response string) bool {
	return container.IsNonFatalMinIOServiceAccountErr(exitCode, response)
}
