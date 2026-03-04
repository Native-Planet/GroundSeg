package container

import (
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
)

type MinioRuntime struct {
	OpenFn                      func(string) (*os.File, error)
	StartramSettingsSnapshotFn  func() config.StartramSettings
	ShipSettingsSnapshotFn      func() config.ShipSettings
	BasePathFn                  func() string
	DockerDirFn                 func() string
	ReadFileFn                  func(string) ([]byte, error)
	WriteFileFn                 func(string, []byte, os.FileMode) error
	MkdirAllFn                  func(string, os.FileMode) error
	CreateDefaultMCConfFn       func() error
	StartContainerFn            func(string, string) (structs.ContainerState, error)
	UpdateContainerStateFn      func(string, structs.ContainerState)
	GetContainerRunningStatusFn func(string) (string, error)
	GetLatestContainerInfoFn    func(string) (map[string]string, error)
	GetLatestContainerImageFn   func(string) (string, error)
	LoadUrbitConfigFn           func(string) error
	UrbitConfFn                 func(string) structs.UrbitDocker
	UpdateUrbitFn               func(string, func(*structs.UrbitDocker) error) error
	SetMinIOPasswordFn          func(string, string) error
	GetMinIOPasswordFn          func(string) (string, error)
	RandReadFn                  func([]byte) (int, error)
	ExecCommandFn               func(string, []string) (string, error)
	ExecCommandExitFn           func(string, []string) (string, int, error)
	CopyFileToVolumeFn          func(string, string, string, string, func() (string, error)) error
	VolumeExistsFn              func(string) (bool, error)
	CreateVolumeFn              func(string) error
	SleepFn                     func(time.Duration)
	PollIntervalFn              func() time.Duration
}

func LoadMCWithRuntime(rt MinioRuntime) error {
	if rt.StartramSettingsSnapshotFn == nil || rt.ShipSettingsSnapshotFn == nil {
		return fmt.Errorf("minio runtime requires settings snapshot callbacks")
	}
	if rt.StartramSettingsSnapshotFn().WgRegistered {
		zap.L().Info("Loading MC container")
		if rt.BasePathFn == nil {
			return fmt.Errorf("missing base path getter")
		}
		confPath := filepath.Join(rt.BasePathFn(), "settings", "mc.json")
		if rt.StartContainerFn == nil || rt.UpdateContainerStateFn == nil {
			return fmt.Errorf("missing container runtime")
		}
		if err := RunContainerWithRuntime(ContainerRuntimePlan{
			ContainerName:         "mc",
			ContainerImage:        "miniomc",
			ConfigPath:            confPath,
			OpenConfigFn:          rt.OpenFn,
			CreateDefaultConfigFn: rt.CreateDefaultMCConfFn,
			StartContainerFn:      rt.StartContainerFn,
			UpdateContainerState:  rt.UpdateContainerStateFn,
		}); err != nil {
			return err
		}
	}
	return nil
}

func LoadMinIOsWithRuntime(rt MinioRuntime) error {
	if rt.StartramSettingsSnapshotFn == nil || rt.ShipSettingsSnapshotFn == nil {
		return fmt.Errorf("minio runtime requires settings snapshot callbacks")
	}
	if rt.StartramSettingsSnapshotFn().WgRegistered {
		zap.L().Info("Loading MinIO containers")
		if rt.StartContainerFn == nil {
			return fmt.Errorf("missing container runtime")
		}
		for _, pier := range rt.ShipSettingsSnapshotFn().Piers {
			label := "minio_" + pier
			info, err := rt.StartContainerFn(label, "minio")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error starting %s Minio: %v", pier, err))
				continue
			}
			if rt.UpdateContainerStateFn != nil {
				rt.UpdateContainerStateFn(label, info)
			}
		}
	}
	return nil
}

func MinioContainerConfWithRuntime(rt MinioRuntime, containerName string) (container.Config, container.HostConfig, error) {
	if rt.GetLatestContainerInfoFn == nil || rt.LoadUrbitConfigFn == nil || rt.RandReadFn == nil || rt.SetMinIOPasswordFn == nil || rt.UrbitConfFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("minio runtime not fully configured")
	}
	var containerConfig container.Config
	var hostConfig container.HostConfig
	shipName, err := GetPatpFromMinIOName(containerName)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	if err := rt.LoadUrbitConfigFn(shipName); err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Error loading %s config: %v", shipName, err)
	}
	shipConf := rt.UrbitConfFn(shipName)
	containerInfo, err := rt.GetLatestContainerInfoFn("minio")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])

	randomBytes := make([]byte, 16)
	if _, err := rt.RandReadFn(randomBytes); err != nil {
		return containerConfig, hostConfig, err
	}
	minIOPwd := hex.EncodeToString(randomBytes)
	if err := rt.SetMinIOPasswordFn(containerName, minIOPwd); err != nil {
		return containerConfig, hostConfig, err
	}

	environment := []string{
		fmt.Sprintf("MINIO_ROOT_USER=%s", shipName),
		fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", minIOPwd),
		fmt.Sprintf("MINIO_DOMAIN=s3.%s", shipConf.WgURL),
		fmt.Sprintf("MINIO_SERVER_URL=https://s3.%s", shipConf.WgURL),
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: containerName,
			Target: "/data",
		},
	}
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{"minio"},
		Cmd: []string{
			"server",
			"/data",
			"--address",
			fmt.Sprintf(":%v", shipConf.WgS3Port),
			"--console-address",
			fmt.Sprintf(":%v", shipConf.WgConsolePort),
		},
		Env: environment,
	}
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
		Mounts:      mounts,
	}
	if err := waitForContainerRunningWithRuntime(rt, "wireguard", 30*time.Second); err != nil {
		return containerConfig, hostConfig, err
	}
	return containerConfig, hostConfig, nil
}

func MCContainerConfWithRuntime(rt MinioRuntime) (container.Config, container.HostConfig, error) {
	if rt.GetLatestContainerInfoFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("minio runtime not fully configured")
	}
	var containerConfig container.Config
	var hostConfig container.HostConfig
	containerInfo, err := rt.GetLatestContainerInfoFn("miniomc")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{"/bin/bash"},
		Tty:        true,
		OpenStdin:  true,
	}
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
	}
	if err := waitForContainerRunningWithRuntime(rt, "wireguard", 30*time.Second); err != nil {
		return containerConfig, hostConfig, err
	}
	return containerConfig, hostConfig, nil
}

func SetMinIOAdminAccountWithRuntime(rt MinioRuntime, containerName string) error {
	patp, err := GetPatpFromMinIOName(containerName)
	if err != nil {
		return err
	}
	if rt.UrbitConfFn == nil || rt.GetMinIOPasswordFn == nil || rt.GetContainerRunningStatusFn == nil {
		return fmt.Errorf("minio runtime not fully configured")
	}
	urbConf := rt.UrbitConfFn(patp)
	if err := waitForContainerRunningWithRuntime(rt, "mc", 30*time.Second); err != nil {
		return err
	}
	pwd, err := rt.GetMinIOPasswordFn(fmt.Sprintf("minio_%s", patp))
	if err != nil {
		return err
	}
	aliasCommand := []string{"mc", "alias", "set", fmt.Sprintf("patp_%s", patp), fmt.Sprintf("http://localhost:%v", urbConf.WgS3Port), patp, pwd}
	if _, _, err := execMinIODockerCommandWithRuntime(rt, containerName, aliasCommand); err != nil {
		return err
	}
	createCommand := []string{"mc", "mb", "--ignore-existing", fmt.Sprintf("patp_%s/bucket", patp)}
	if _, _, err := execMinIODockerCommandWithRuntime(rt, containerName, createCommand); err != nil {
		return err
	}

	scriptContent := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, "bucket")
	if rt.DockerDirFn == nil {
		return fmt.Errorf("missing docker dir getter")
	}
	scriptPath := filepath.Join(rt.DockerDirFn(), containerName, "_data", "policy.json")
	ensureVolumes := artifactwriter.NewVolumeInitializationPlan(artifactwriter.VolumeOps{
		VolumeExistsFn: rt.VolumeExistsFn,
		CreateVolumeFn: rt.CreateVolumeFn,
	}, containerName).EnsureVolumes
	err = artifactwriter.Write(artifactwriter.WriteConfig{
		FilePath:            scriptPath,
		Content:             scriptContent,
		FileMode:            0755,
		DirectoryMode:       0755,
		WriteFileFn:         rt.WriteFileFn,
		MkdirAllFn:          rt.MkdirAllFn,
		CopyToVolumeFn:      rt.CopyFileToVolumeFn,
		TargetPath:          "/data/",
		VolumeName:          containerName,
		WriterContainerName: "mc_writer",
		SelectImageFn: func() (string, error) {
			if rt.GetLatestContainerImageFn == nil {
				return "", fmt.Errorf("missing image selector")
			}
			return rt.GetLatestContainerImageFn("miniomc")
		},
		CopyErrorPrefix: "Failed to copy policy file to volume",
		EnsureVolumesFn: ensureVolumes,
	})
	if err != nil {
		return err
	}
	policyCommand := []string{"mc", "anonymous", "set-json", "/data/policy.json", fmt.Sprintf("patp_%s/bucket", patp)}
	if _, _, err := execMinIODockerCommandWithRuntime(rt, containerName, policyCommand); err != nil {
		return err
	}
	return nil
}

func execMinIODockerCommandWithRuntime(rt MinioRuntime, containerName string, command []string) (string, int, error) {
	if rt.ExecCommandExitFn == nil {
		return "", 0, fmt.Errorf("missing command executor")
	}
	return rt.ExecCommandExitFn(containerName, command)
}

func IsNonFatalMinIOServiceAccountErr(exitCode int, response string) bool {
	if exitCode == 0 {
		return false
	}
	if exitCode == 1 {
		return true
	}
	return strings.Contains(strings.ToLower(response), "no such access key") || strings.Contains(strings.ToLower(response), "not found")
}

func waitForContainerRunningWithRuntime(rt MinioRuntime, containerName string, timeout time.Duration) error {
	if rt.GetContainerRunningStatusFn == nil || rt.SleepFn == nil || rt.PollIntervalFn == nil {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for {
		status, err := rt.GetContainerRunningStatusFn(containerName)
		if err == nil && strings.Contains(status, "Up") {
			return nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				return fmt.Errorf("container %s did not reach running state within %s", containerName, timeout)
			}
			return fmt.Errorf("error while waiting for container %s to reach running state: %w", containerName, err)
		}
		rt.SleepFn(rt.PollIntervalFn())
	}
}

func GetPatpFromMinIOName(containerName string) (string, error) {
	if !strings.HasPrefix(containerName, "minio_") {
		return "", fmt.Errorf("Invalid MinIO container name")
	}
	splitStr := strings.SplitN(containerName, "_", 2)
	return splitStr[1], nil
}

func CreateMinIOServiceAccountWithRuntime(rt MinioRuntime, patp string) (structs.MinIOServiceAccount, error) {
	var svcAccount structs.MinIOServiceAccount
	svcAccount.AccessKey = "urbit_minio"
	svcAccount.Alias = fmt.Sprintf("patp_%s", patp)
	svcAccount.User = patp
	bytes := make([]byte, 20)
	if rt.RandReadFn == nil {
		return svcAccount, fmt.Errorf("missing random source")
	}
	_, err := rt.RandReadFn(bytes)
	if err != nil {
		return svcAccount, err
	}
	svcAccount.SecretKey = hex.EncodeToString(bytes)
	containerName := fmt.Sprintf("minio_%s", patp)
	cmd := []string{"mc", "admin", "user", "svcacct", "rm", svcAccount.Alias, fmt.Sprintf("%s", svcAccount.AccessKey)}
	response, exitCode, err := execMinIODockerCommandWithRuntime(rt, containerName, cmd)
	if err != nil {
		if !IsNonFatalMinIOServiceAccountErr(exitCode, response) {
			return svcAccount, fmt.Errorf("Failed to remove old service account for %s: %v", patp, err)
		}
		zap.L().Warn(fmt.Sprintf("Ignoring non-fatal service account remove error for %s: %v", patp, err))
	} else {
		zap.L().Debug(fmt.Sprintf("Remove old service account response: %s", response))
	}
	response, _, err = execMinIODockerCommandWithRuntime(rt, containerName, []string{
		"mc",
		"admin",
		"user",
		"svcacct",
		"add",
		"--access-key",
		fmt.Sprintf("%s", svcAccount.AccessKey),
		"--secret-key",
		fmt.Sprintf("%s", svcAccount.SecretKey),
		svcAccount.Alias,
		svcAccount.User,
	})
	if err != nil {
		return svcAccount, fmt.Errorf("Failed to create service account for %s: %v", patp, err)
	}

	return svcAccount, nil
}

// NOTE: no deprecated aliases required here; orchestration wrappers use
// package-level functions.
