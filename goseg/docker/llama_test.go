package docker

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestLoadLlamaDisabledNoop(t *testing.T) {
	t.Cleanup(func() {
		confForLlama = config.Conf
		stopContainerByNameForLlama = StopContainerByName
		startContainerForLlama = StartContainer
		updateContainerStateForLlama = config.UpdateContainerState
	})

	started := false
	stopped := false
	updated := false
	confForLlama = func() structs.SysConfig {
		return structs.SysConfig{PenpaiAllow: false}
	}
	stopContainerByNameForLlama = func(string) error {
		stopped = true
		return nil
	}
	startContainerForLlama = func(string, string) (structs.ContainerState, error) {
		started = true
		return structs.ContainerState{}, nil
	}
	updateContainerStateForLlama = func(string, structs.ContainerState) {
		updated = true
	}

	if err := LoadLlama(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if started || stopped || updated {
		t.Fatalf("expected no docker operations when penpai disabled")
	}
}

func TestLoadLlamaStartsAndUpdatesState(t *testing.T) {
	t.Cleanup(func() {
		confForLlama = config.Conf
		stopContainerByNameForLlama = StopContainerByName
		startContainerForLlama = StartContainer
		updateContainerStateForLlama = config.UpdateContainerState
	})

	var stoppedName string
	var startedName, startedType string
	var updatedName string
	var updatedState structs.ContainerState

	confForLlama = func() structs.SysConfig {
		return structs.SysConfig{PenpaiAllow: true, PenpaiRunning: false}
	}
	stopContainerByNameForLlama = func(name string) error {
		stoppedName = name
		return nil
	}
	startContainerForLlama = func(name, containerType string) (structs.ContainerState, error) {
		startedName, startedType = name, containerType
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updateContainerStateForLlama = func(name string, state structs.ContainerState) {
		updatedName = name
		updatedState = state
	}

	if err := LoadLlama(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stoppedName != "llama-gpt-api" {
		t.Fatalf("unexpected stop call: %s", stoppedName)
	}
	if startedName != "llama-gpt-api" || startedType != "llama-api" {
		t.Fatalf("unexpected start call: %s %s", startedName, startedType)
	}
	if updatedName != "llama-api" || updatedState.ActualStatus != "running" {
		t.Fatalf("unexpected update call: name=%s state=%+v", updatedName, updatedState)
	}
}

func TestLoadLlamaStartError(t *testing.T) {
	t.Cleanup(func() {
		confForLlama = config.Conf
		startContainerForLlama = StartContainer
	})

	confForLlama = func() structs.SysConfig {
		return structs.SysConfig{PenpaiAllow: true, PenpaiRunning: true}
	}
	startContainerForLlama = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, errors.New("start failed")
	}

	err := LoadLlama()
	if err == nil || !strings.Contains(err.Error(), "Error starting Llama API") {
		t.Fatalf("expected wrapped start error, got %v", err)
	}
}

func TestLlamaApiContainerConfBuildsExpectedConfig(t *testing.T) {
	t.Cleanup(func() {
		confForLlama = config.Conf
		volumeExistsForLlama = volumeExists
		createVolumeForLlama = CreateVolume
		addOrGetNetworkForLlama = addOrGetNetwork
		writeFileForLlama = ioutil.WriteFile
	})

	oldDockerDir := config.DockerDir
	oldVolumeDir := VolumeDir
	oldUrbitConfig := config.UrbitsConfig
	config.DockerDir = "/tmp/docker-data"
	VolumeDir = "/tmp/volume-data"
	t.Cleanup(func() {
		config.DockerDir = oldDockerDir
		VolumeDir = oldVolumeDir
		config.UrbitsConfig = oldUrbitConfig
	})

	config.UrbitsConfig = map[string]structs.UrbitDocker{
		"~zod": {BootStatus: "boot"},
		"~bus": {BootStatus: "stopped"},
	}

	var volumeChecks []string
	volumeExistsForLlama = func(name string) (bool, error) {
		volumeChecks = append(volumeChecks, name)
		return false, nil
	}
	var createdVolumes []string
	createVolumeForLlama = func(name string) error {
		createdVolumes = append(createdVolumes, name)
		return nil
	}
	addOrGetNetworkForLlama = func(string) (string, error) {
		return "llama-net", nil
	}
	var gotScriptPath string
	writeFileForLlama = func(path string, _ []byte, _ os.FileMode) error {
		gotScriptPath = path
		return nil
	}

	confForLlama = func() structs.SysConfig {
		return structs.SysConfig{
			Piers:        []string{"~zod", "~bus"},
			PenpaiActive: "phi.gguf",
			PenpaiModels: []structs.Penpai{
				{ModelName: "phi.gguf", ModelUrl: "https://models.example/phi.gguf"},
			},
		}
	}

	containerCfg, hostCfg, err := llamaApiContainerConf()
	if err != nil {
		t.Fatalf("llamaApiContainerConf failed: %v", err)
	}
	if !reflect.DeepEqual(volumeChecks, []string{"llama-gpt-api", "llama-gpt-api_api"}) {
		t.Fatalf("unexpected volume checks: %v", volumeChecks)
	}
	if !reflect.DeepEqual(createdVolumes, []string{"llama-gpt-api", "llama-gpt-api_api"}) {
		t.Fatalf("unexpected created volumes: %v", createdVolumes)
	}
	wantScriptPath := filepath.Join(config.DockerDir, "llama-gpt-api_api", "_data", "run.sh")
	if gotScriptPath != wantScriptPath {
		t.Fatalf("unexpected script path: got %s want %s", gotScriptPath, wantScriptPath)
	}
	if containerCfg.Image == "" || containerCfg.Hostname != "llama-gpt-api" {
		t.Fatalf("unexpected container config: %+v", containerCfg)
	}
	env := strings.Join(containerCfg.Env, " ")
	if !strings.Contains(env, "MODEL=/models/phi.gguf") || !strings.Contains(env, "MODEL_DOWNLOAD_URL=https://models.example/phi.gguf") {
		t.Fatalf("unexpected env values: %v", containerCfg.Env)
	}
	if string(hostCfg.NetworkMode) != "llama-net" {
		t.Fatalf("unexpected network mode: %s", hostCfg.NetworkMode)
	}
	if len(hostCfg.Binds) != 1 || !strings.Contains(hostCfg.Binds[0], "/piers/~zod") {
		t.Fatalf("expected bind only for booted pier, got %v", hostCfg.Binds)
	}
}

func TestLlamaApiContainerConfErrorsWhenActiveModelMissing(t *testing.T) {
	t.Cleanup(func() {
		confForLlama = config.Conf
		volumeExistsForLlama = volumeExists
		addOrGetNetworkForLlama = addOrGetNetwork
		writeFileForLlama = ioutil.WriteFile
	})

	volumeExistsForLlama = func(string) (bool, error) { return true, nil }
	addOrGetNetworkForLlama = func(string) (string, error) { return "llama-net", nil }
	writeFileForLlama = func(string, []byte, os.FileMode) error { return nil }
	confForLlama = func() structs.SysConfig {
		return structs.SysConfig{
			PenpaiActive: "missing-model",
			PenpaiModels: []structs.Penpai{{ModelName: "other-model", ModelUrl: "url"}},
		}
	}

	_, _, err := llamaApiContainerConf()
	if err == nil || !strings.Contains(err.Error(), "active penpai model") {
		t.Fatalf("expected missing model error, got %v", err)
	}
}
