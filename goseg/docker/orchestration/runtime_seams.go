package orchestration

import (
	"crypto/rand"
	"groundseg/config"
	"groundseg/internal/seams"
	"groundseg/structs"
	"os"
	"time"
)

type dockerRuntimeBase struct {
	seams.DockerRuntimeBase
}

func newDockerRuntimeBase() dockerRuntimeBase {
	return dockerRuntimeBase{DockerRuntimeBase: seams.NewDockerRuntimeBase()}
}

type dockerRuntime struct {
	dockerRuntimeBase

	osOpen                            func(string) (*os.File, error)
	createDefaultWGConf               func() error
	createDefaultNetdataConf          func() error
	createDefaultMcConf               func() error
	writeWgConf                       func(dockerRuntime) error
	writeNDConf                       func(dockerRuntime) error
	startContainer                    func(string, string) (structs.ContainerState, error)
	updateContainerState              func(string, structs.ContainerState)
	getLatestContainerInfo            func(string) (map[string]string, error)
	getWgConf                         func() (structs.WgConfig, error)
	getStartramConfig                 func() structs.StartramRetrieve
	conf                              func() structs.SysConfig
	readFile                          func(string) ([]byte, error)
	writeFile                         func(string, []byte, os.FileMode) error
	mkdirAll                          func(string, os.FileMode) error
	copyWGFileToVolume                func(dockerRuntime, string, string, string) error
	copyNDFileToVolume                func(dockerRuntime, string, string, string) error
	copyFileToVolumeWithTempContainer func(string, string, string, string, volumeWriterImageSelector) error
	latestContainerImage              func(string) (string, error)
	shipSettings                      func() config.ShipSettings
	runtimeSettings                   func() config.ShipRuntimeSettings
	loadUrbitConfig                   func(string) error
	urbitConf                         func(string) structs.UrbitDocker
	setMinIOPassword                  func(string, string) error
	createContainer                   func(string, string) (structs.ContainerState, error)
	updateUrbit                       func(string, func(*structs.UrbitDocker) error) error
	getContainerRunningStatus         func(string) (string, error)
	sleep                             func(time.Duration)
	execDockerCommand                 func(string, []string) (string, error)
	execDockerCommandWithExitCode     func(string, []string) (string, int, error)
	getMinIOPassword                  func(string) (string, error)
	pollInterval                      time.Duration
	randRead                          func([]byte) (int, error)
}

type wireguardRuntime = dockerRuntime
type netdataRuntime = dockerRuntime
type minioRuntime = dockerRuntime
type urbitRuntime = dockerRuntime

func newDockerRuntime() dockerRuntime {
	base := newDockerRuntimeBase()
	return dockerRuntime{
		dockerRuntimeBase:                 base,
		osOpen:                            os.Open,
		startContainer:                    StartContainer,
		updateContainerState:              config.UpdateContainerState,
		getLatestContainerInfo:            GetLatestContainerInfo,
		readFile:                          os.ReadFile,
		writeFile:                         os.WriteFile,
		mkdirAll:                          os.MkdirAll,
		copyFileToVolumeWithTempContainer: copyFileToVolumeWithTempContainer,
		latestContainerImage:              latestContainerImage,
	}
}

func newWireguardRuntime() wireguardRuntime {
	rt := newDockerRuntime()
	rt.createDefaultWGConf = config.CreateDefaultWGConf
	rt.getWgConf = config.GetWgConf
	rt.getStartramConfig = config.GetStartramConfig
	rt.conf = config.Conf
	return rt
}

func newNetdataRuntime() netdataRuntime {
	rt := newDockerRuntime()
	rt.createDefaultNetdataConf = config.CreateDefaultNetdataConf
	return rt
}

func newMinIORuntime() minioRuntime {
	rt := newDockerRuntime()
	rt.conf = config.Conf
	rt.createDefaultMcConf = config.CreateDefaultMcConf
	rt.loadUrbitConfig = config.LoadUrbitConfig
	rt.urbitConf = config.UrbitConf
	rt.setMinIOPassword = config.SetMinIOPassword
	rt.getContainerRunningStatus = GetContainerRunningStatus
	rt.sleep = time.Sleep
	rt.execDockerCommand = func(containerName string, cmd []string) (string, error) {
		output, _, err := ExecDockerCommand(containerName, cmd)
		return output, err
	}
	rt.execDockerCommandWithExitCode = ExecDockerCommand
	rt.getMinIOPassword = config.GetMinIOPassword
	rt.pollInterval = 500 * time.Millisecond
	rt.randRead = rand.Read
	return rt
}

func newUrbitRuntime() urbitRuntime {
	rt := newDockerRuntime()
	rt.shipSettings = config.ShipSettingsSnapshot
	rt.runtimeSettings = config.ShipRuntimeSettingsSnapshot
	rt.loadUrbitConfig = config.LoadUrbitConfig
	rt.urbitConf = config.UrbitConf
	rt.createContainer = CreateContainer
	rt.updateUrbit = config.UpdateUrbit
	return rt
}
