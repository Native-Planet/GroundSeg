package orchestration

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func testLlamaRuntime(dockerDir string) dockerRuntime {
	return dockerRuntime{
		contextOps: RuntimeContextOps{
			DockerDirFn: func() string { return dockerDir },
		},
		fileOps: RuntimeFileOps{
			WriteFileFn: func(string, []byte, os.FileMode) error { return nil },
		},
		containerOps: RuntimeContainerOps{
			StopContainerByNameFn:  func(string) error { return nil },
			StartContainerFn:       func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
			CreateContainerFn:      nil,
			UpdateContainerStateFn: func(string, structs.ContainerState) {},
			AddOrGetNetworkFn:      func(string) (string, error) { return "default", nil },
		},
		configOps: RuntimeSnapshotOps{
			StartramSettingsSnapshotFn: func() config.StartramSettings { return config.StartramSettings{} },
			PenpaiSettingsSnapshotFn:   func() config.PenpaiSettings { return config.PenpaiSettings{} },
			ShipSettingsSnapshotFn:     func() config.ShipSettings { return config.ShipSettings{} },
		},
		urbitOps: RuntimeUrbitOps{
			UrbitConfAllFn: func() map[string]structs.UrbitDocker { return map[string]structs.UrbitDocker{} },
		},
		volumeOps: RuntimeVolumeOps{
			VolumeExistsFn: func(string) (bool, error) { return false, nil },
			CreateVolumeFn: func(string) error { return nil },
		},
	}
}

func TestLoadLlamaDisabledNoop(t *testing.T) {
	var stopped, started, updated bool

	rt := testLlamaRuntime(t.TempDir())
	rt.configOps.PenpaiSettingsSnapshotFn = func() config.PenpaiSettings {
		return config.PenpaiSettings{Allowed: false}
	}
	rt.containerOps.StopContainerByNameFn = func(string) error {
		stopped = true
		return nil
	}
	rt.containerOps.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		started = true
		return structs.ContainerState{}, nil
	}
	rt.containerOps.UpdateContainerStateFn = func(string, structs.ContainerState) {
		updated = true
	}

	if err := loadLlamaWithRuntime(llamaRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if started || stopped || updated {
		t.Fatalf("expected no docker operations when penpai disabled")
	}
}

func TestLoadLlamaStartsAndUpdatesState(t *testing.T) {
	var stoppedName string
	var startedName, startedType string
	var updatedName string
	var updatedState structs.ContainerState

	rt := testLlamaRuntime(t.TempDir())
	rt.configOps.PenpaiSettingsSnapshotFn = func() config.PenpaiSettings {
		return config.PenpaiSettings{
			Allowed: true,
			Running: false,
		}
	}
	rt.containerOps.StopContainerByNameFn = func(name string) error {
		stoppedName = name
		return nil
	}
	rt.containerOps.StartContainerFn = func(name, containerType string) (structs.ContainerState, error) {
		startedName, startedType = name, containerType
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	rt.containerOps.UpdateContainerStateFn = func(name string, state structs.ContainerState) {
		updatedName = name
		updatedState = state
	}

	if err := loadLlamaWithRuntime(llamaRuntimeFromDocker(rt)); err != nil {
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
	rt := testLlamaRuntime(t.TempDir())
	rt.configOps.PenpaiSettingsSnapshotFn = func() config.PenpaiSettings {
		return config.PenpaiSettings{
			Allowed: true,
			Running: true,
		}
	}
	rt.containerOps.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, errors.New("start failed")
	}

	err := loadLlamaWithRuntime(llamaRuntimeFromDocker(rt))
	if err == nil || !strings.Contains(err.Error(), "Error starting Llama API") {
		t.Fatalf("expected wrapped start error, got %v", err)
	}
}

func TestLlamaApiContainerConfBuildsExpectedConfig(t *testing.T) {
	dockerDir := t.TempDir()
	rt := testLlamaRuntime(dockerDir)
	urbitConfig := map[string]structs.UrbitDocker{
		"~zod": {
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				BootStatus: "boot",
			},
		},
		"~bus": {
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				BootStatus: "stopped",
			},
		},
	}
	rt.urbitOps.UrbitConfAllFn = func() map[string]structs.UrbitDocker {
		return urbitConfig
	}

	var volumeChecks []string
	rt.volumeOps.VolumeExistsFn = func(name string) (bool, error) {
		volumeChecks = append(volumeChecks, name)
		return false, nil
	}
	var createdVolumes []string
	rt.volumeOps.CreateVolumeFn = func(name string) error {
		createdVolumes = append(createdVolumes, name)
		return nil
	}
	rt.containerOps.AddOrGetNetworkFn = func(string) (string, error) {
		return "llama-net", nil
	}
	var gotScriptPath string
	rt.fileOps.WriteFileFn = func(path string, _ []byte, _ os.FileMode) error {
		gotScriptPath = path
		return nil
	}

	rt.configOps.ShipSettingsSnapshotFn = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"~zod", "~bus"}}
	}
	rt.configOps.PenpaiSettingsSnapshotFn = func() config.PenpaiSettings {
		return config.PenpaiSettings{
			ActiveModel: "phi.gguf",
			Models: []structs.Penpai{
				{ModelName: "phi.gguf", ModelUrl: "https://models.example/phi.gguf"},
			},
		}
	}

	containerCfg, hostCfg, err := llamaApiContainerConfWithRuntime(llamaRuntimeFromDocker(rt))
	if err != nil {
		t.Fatalf("llamaApiContainerConf failed: %v", err)
	}
	if !reflect.DeepEqual(volumeChecks, []string{"llama-gpt-api", "llama-gpt-api_api"}) {
		t.Fatalf("unexpected volume checks: %v", volumeChecks)
	}
	if !reflect.DeepEqual(createdVolumes, []string{"llama-gpt-api", "llama-gpt-api_api"}) {
		t.Fatalf("unexpected created volumes: %v", createdVolumes)
	}
	wantScriptPath := filepath.Join(dockerDir, "llama-gpt-api_api", "_data", "run.sh")
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
	rt := testLlamaRuntime(t.TempDir())
	rt.volumeOps.VolumeExistsFn = func(string) (bool, error) { return true, nil }
	rt.containerOps.AddOrGetNetworkFn = func(string) (string, error) { return "llama-net", nil }
	rt.fileOps.WriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	rt.configOps.PenpaiSettingsSnapshotFn = func() config.PenpaiSettings {
		return config.PenpaiSettings{
			ActiveModel: "missing-model",
			Models:      []structs.Penpai{{ModelName: "other-model", ModelUrl: "url"}},
		}
	}
	rt.configOps.ShipSettingsSnapshotFn = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"~zod"}}
	}

	_, _, err := llamaApiContainerConfWithRuntime(llamaRuntimeFromDocker(rt))
	if err == nil || !strings.Contains(err.Error(), "active penpai model") {
		t.Fatalf("expected missing model error, got %v", err)
	}
}
