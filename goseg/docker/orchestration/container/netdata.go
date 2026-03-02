package container

import (
	"fmt"
	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/structs"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

type NetdataRuntime struct {
	OpenFn                    func(string) (*os.File, error)
	ReadFileFn                func(string) ([]byte, error)
	WriteFileFn               func(string, []byte, os.FileMode) error
	MkdirAllFn                func(string, os.FileMode) error
	StartContainerFn          func(string, string) (structs.ContainerState, error)
	UpdateContainerState      func(string, structs.ContainerState)
	CreateDefaultFn           func() error
	WriteNDConfFn             func() error
	GetLatestContainerInfoFn  func(string) (map[string]string, error)
	GetLatestContainerImageFn func(string) (string, error)
	CopyFileToVolumeFn        func(string, string, string, string, func() (string, error)) error
	VolumeExistsFn            func(string) (bool, error)
	CreateVolumeFn            func(string) error
	DockerDirFn               func() string

	BasePathFn                  func() string
	GetContainerRunningStatusFn func(string) (string, error)
	SleepFn                     func(time.Duration)
	PollIntervalFn              func() time.Duration
}

func LoadNetdataWithRuntime(rt NetdataRuntime) error {
	zap.L().Info("Loading NetData container")
	if rt.BasePathFn == nil {
		return fmt.Errorf("missing base path getter")
	}
	confPath := filepath.Join(rt.BasePathFn(), "settings", "netdata.json")
	writeConf := func() error {
		if rt.WriteNDConfFn != nil {
			return rt.WriteNDConfFn()
		}
		return WriteNDConfWithRuntime(rt)
	}
	err := RunContainerWithRuntime(ContainerRuntimePlan{
		ContainerName:         "netdata",
		ContainerImage:        "netdata",
		ConfigPath:            confPath,
		OpenConfigFn:          rt.OpenFn,
		CreateDefaultConfigFn: rt.CreateDefaultFn,
		WriteConfigFn:         writeConf,
		StartContainerFn:      rt.StartContainerFn,
		UpdateContainerState:  rt.UpdateContainerState,
	})
	if err != nil {
		return err
	}
	return nil
}

func NetdataContainerConfWithRuntime(rt NetdataRuntime) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	if rt.GetLatestContainerInfoFn == nil {
		return containerConfig, hostConfig, fmt.Errorf("missing latest netdata metadata runtime")
	}
	containerInfo, err := rt.GetLatestContainerInfoFn("netdata")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("lookup latest netdata metadata: %w", err)
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	containerConfig = container.Config{
		Image:        desiredImage,
		ExposedPorts: nat.PortSet{"19999/tcp": struct{}{}},
		Volumes: map[string]struct{}{
			"/etc/netdata":         {},
			"/var/lib/netdata":     {},
			"/var/cache/netdata":   {},
			"/host/etc/passwd":     {},
			"/host/etc/group":      {},
			"/host/proc":           {},
			"/host/sys":            {},
			"/host/etc/os-release": {},
		},
	}
	hostConfig = container.HostConfig{
		CapAdd: []string{"SYS_PTRACE"},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			NanoCPUs: 1e8,
			Memory:   200 * 1024 * 1024,
		},
		SecurityOpt: []string{"apparmor=unconfined"},
		PortBindings: nat.PortMap{
			"19999/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "19999"}},
		},
		Binds: []string{
			"netdataconfig:/etc/netdata",
			"netdatalib:/var/lib/netdata",
			"netdatacache:/var/cache/netdata",
			"/etc/passwd:/host/etc/passwd:ro",
			"/etc/group:/host/etc/group:ro",
			"/proc:/host/proc:ro",
			"/sys:/host/sys:ro",
			"/etc/os-release:/host/etc/os-release:ro",
		},
	}
	return containerConfig, hostConfig, nil
}

func WriteNDConfWithRuntime(rt NetdataRuntime) error {
	newConf := "[plugins]\n     apps = no\n"
	if rt.DockerDirFn == nil {
		return fmt.Errorf("missing docker dir getter")
	}
	filePath := filepath.Join(rt.DockerDirFn(), "netdataconfig", "_data", "netdata.conf")
	if rt.ReadFileFn == nil {
		return fmt.Errorf("missing file reader")
	}
	existingConf, err := rt.ReadFileFn(filePath)
	if err != nil {
		zap.L().Info("Creating ND config")
		return writeNDConfigArtifactWithRuntime(rt, filePath, newConf)
	}
	if string(existingConf) != newConf {
		zap.L().Info("Writing ND config")
		return writeNDConfigArtifactWithRuntime(rt, filePath, newConf)
	}
	return nil
}

func writeNDConfigArtifactWithRuntime(rt NetdataRuntime, filePath string, content string) error {
	return artifactwriter.Write(artifactwriter.WriteConfig{
		FilePath:            filePath,
		Content:             content,
		FileMode:            0644,
		DirectoryMode:       0755,
		WriteFileFn:         rt.WriteFileFn,
		MkdirAllFn:          rt.MkdirAllFn,
		CopyToVolumeFn:      rt.CopyFileToVolumeFn,
		TargetPath:          "/etc/netdata/",
		VolumeName:          "netdata",
		WriterContainerName: "nd_writer",
		SelectImageFn: func() (string, error) {
			if rt.GetLatestContainerImageFn == nil {
				return "", fmt.Errorf("missing image selector")
			}
			return rt.GetLatestContainerImageFn("netdata")
		},
		CopyErrorPrefix: "Failed to copy ND config file to volume",
		EnsureVolumesFn: artifactwriter.NewVolumeInitializationPlan(artifactwriter.VolumeOps{
			VolumeExistsFn: rt.VolumeExistsFn,
			CreateVolumeFn: rt.CreateVolumeFn,
		}, "netdata").EnsureVolumes,
	})
}

func WriteNDConfToFileWithRuntime(rt NetdataRuntime, filePath string, content string) error {
	return writeNDConfigArtifactWithRuntime(rt, filePath, content)
}

func CopyNDFileToVolume(filePath string, targetPath string, volumeName string) error {
	return CopyNDFileToVolumeWithRuntime(NetdataRuntime{}, filePath, targetPath, volumeName)
}

func CopyNDFileToVolumeWithRuntime(rt NetdataRuntime, filePath string, targetPath string, volumeName string) error {
	if rt.CopyFileToVolumeFn == nil {
		return fmt.Errorf("missing copy-to-volume runtime")
	}
	return rt.CopyFileToVolumeFn(
		filePath,
		targetPath,
		volumeName,
		"nd_writer",
		func() (string, error) {
			if rt.GetLatestContainerImageFn == nil {
				return "", fmt.Errorf("missing image selector")
			}
			return rt.GetLatestContainerImageFn("netdata")
		},
	)
}
