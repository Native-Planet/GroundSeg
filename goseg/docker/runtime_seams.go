package docker

import (
	"crypto/rand"
	"groundseg/config"
	"groundseg/structs"
	"net/http"
	"os"
	"time"
)

type dockerRuntime struct {
	osOpen                            func(string) (*os.File, error)
	createDefaultWGConf               func() error
	createDefaultNetdataConf          func() error
	createDefaultMcConf               func() error
	writeWgConf                       func(dockerRuntime) error
	writeNDConf                       func(dockerRuntime) error
	startContainer                    func(string, string) (structs.ContainerState, error)
	createContainer                   func(string, string) (structs.ContainerState, error)
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
	loadUrbitConfig                   func(string) error
	urbitConf                         func(string) structs.UrbitDocker
	randRead                          func([]byte) (int, error)
	setMinIOPassword                  func(string, string) error
	getContainerRunningStatus         func(string) (string, error)
	sleep                             func(time.Duration)
	execDockerCommand                 func(string, []string) (string, error)
	getMinIOPassword                  func(string) (string, error)
	pollInterval                      time.Duration
	shipSettings                      func() config.ShipSettings
	runtimeSettings                   func() config.ShipRuntimeSettings
	updateUrbit                       func(string, func(*structs.UrbitDocker) error) error
	architecture                      func() string
	basePath                          func() string
	dockerDir                         func() string
	httpGet                           func(string) (*http.Response, error)
}

func newDockerRuntime() dockerRuntime {
	return dockerRuntime{
		osOpen:                            os.Open,
		createDefaultWGConf:               config.CreateDefaultWGConf,
		createDefaultNetdataConf:          config.CreateDefaultNetdataConf,
		createDefaultMcConf:               config.CreateDefaultMcConf,
		writeWgConf:                       writeWgConfWithRuntime,
		writeNDConf:                       writeNDConfWithRuntime,
		startContainer:                    StartContainer,
		createContainer:                   CreateContainer,
		updateContainerState:              config.UpdateContainerState,
		getLatestContainerInfo:            GetLatestContainerInfo,
		getWgConf:                         config.GetWgConf,
		getStartramConfig:                 config.GetStartramConfig,
		conf:                              config.Conf,
		readFile:                          os.ReadFile,
		writeFile:                         os.WriteFile,
		mkdirAll:                          os.MkdirAll,
		copyWGFileToVolume:                copyWGFileToVolumeWithRuntime,
		copyNDFileToVolume:                copyNDFileToVolumeWithRuntime,
		copyFileToVolumeWithTempContainer: copyFileToVolumeWithTempContainer,
		latestContainerImage:              latestContainerImage,
		loadUrbitConfig:                   config.LoadUrbitConfig,
		urbitConf:                         config.UrbitConf,
		randRead:                          rand.Read,
		setMinIOPassword:                  config.SetMinIOPassword,
		getContainerRunningStatus:         GetContainerRunningStatus,
		sleep:                             time.Sleep,
		execDockerCommand:                 ExecDockerCommand,
		getMinIOPassword:                  config.GetMinIOPassword,
		pollInterval:                      500 * time.Millisecond,
		shipSettings:                      config.ShipSettingsSnapshot,
		runtimeSettings:                   config.ShipRuntimeSettingsSnapshot,
		updateUrbit:                       config.UpdateUrbit,
		architecture:                      func() string { return config.Architecture },
		dockerDir:                         func() string { return config.DockerDir },
		basePath:                          func() string { return config.BasePath },
		httpGet:                           http.Get,
	}
}
