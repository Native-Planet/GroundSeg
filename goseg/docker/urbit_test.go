package docker

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/defaults"
	"groundseg/structs"

	"github.com/docker/docker/api/types/mount"
)

func testUrbitRuntime() urbitRuntime {
	rt := newUrbitRuntime()
	rt.shipSettings = func() config.ShipSettings { return config.ShipSettings{} }
	rt.runtimeSettings = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{} }
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.startContainer = func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil }
	rt.createContainer = func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil }
	rt.updateContainerState = func(string, structs.ContainerState) {}
	rt.getLatestContainerInfo = func(kind string) (map[string]string, error) {
		return map[string]string{"repo": "repo/" + kind, "tag": "tag", "hash": "hash"}, nil
	}
	rt.updateUrbit = func(string, func(*structs.UrbitDocker) error) error { return nil }
	rt.writeFile = func(string, []byte, os.FileMode) error { return nil }
	rt.architecture = func() string { return "amd64" }
	rt.dockerDir = func() string { return "/tmp/docker" }
	return rt
}

func TestLoadUrbitsDispatchesStartOrCreate(t *testing.T) {
	rt := testUrbitRuntime()
	rt.shipSettings = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"~zod", "~bus", "~mar"}}
	}
	rt.loadUrbitConfig = func(pier string) error {
		if pier == "~bus" {
			return errors.New("bad config")
		}
		return nil
	}
	rt.urbitConf = func(pier string) structs.UrbitDocker {
		if pier == "~mar" {
			return structs.UrbitDocker{BootStatus: "noboot"}
		}
		return structs.UrbitDocker{BootStatus: "boot"}
	}
	var started, created []string
	rt.startContainer = func(name, ctype string) (structs.ContainerState, error) {
		started = append(started, name+":"+ctype)
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	rt.createContainer = func(name, ctype string) (structs.ContainerState, error) {
		created = append(created, name+":"+ctype)
		return structs.ContainerState{ActualStatus: "created"}, nil
	}
	var updated []string
	rt.updateContainerState = func(name string, _ structs.ContainerState) {
		updated = append(updated, name)
	}

	if err := loadUrbits(rt); err != nil {
		t.Fatalf("LoadUrbits failed: %v", err)
	}
	if strings.Join(started, ",") != "~zod:vere" {
		t.Fatalf("unexpected started ships: %v", started)
	}
	if strings.Join(created, ",") != "~mar:vere" {
		t.Fatalf("unexpected created ships: %v", created)
	}
	if strings.Join(updated, ",") != "~zod,~mar" {
		t.Fatalf("unexpected updated states: %v", updated)
	}
}

func TestUrbitContainerConfDefaultNetworkAndPackReset(t *testing.T) {
	rt := testUrbitRuntime()
	rt.runtimeSettings = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{SnapTime: 90} }
	shipState := map[string]structs.UrbitDocker{
		"~zod": {
			PierName:         "~zod",
			BootStatus:       "pack",
			Network:          "bridge",
			HTTPPort:         8080,
			AmesPort:         34344,
			LoomSize:         31,
			UrbitAmd64Sha256: "old-urbit",
			MinioAmd64Sha256: "old-minio",
		},
	}
	rt.urbitConf = func(name string) structs.UrbitDocker { return shipState[name] }
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.getLatestContainerInfo = func(kind string) (map[string]string, error) {
		switch kind {
		case "vere":
			return map[string]string{"repo": "repo/vere", "tag": "v4.0", "hash": "new-urbit"}, nil
		case "minio":
			return map[string]string{"repo": "repo/minio", "tag": "latest", "hash": "new-minio"}, nil
		default:
			return nil, errors.New("unknown kind")
		}
	}
	rt.architecture = func() string { return "amd64" }
	rt.updateUrbit = func(name string, mutate func(*structs.UrbitDocker) error) error {
		c := shipState[name]
		if err := mutate(&c); err != nil {
			return err
		}
		shipState[name] = c
		return nil
	}
	rt.dockerDir = func() string { return "/tmp/docker" }
	var scriptPath, scriptContent string
	rt.writeFile = func(path string, data []byte, _ os.FileMode) error {
		scriptPath = path
		scriptContent = string(data)
		return nil
	}

	containerCfg, hostCfg, err := urbitContainerConfWithRuntime(rt, "~zod")
	if err != nil {
		t.Fatalf("urbitContainerConf failed: %v", err)
	}
	if containerCfg.Image != "repo/vere:v4.0@sha256:new-urbit" {
		t.Fatalf("unexpected image: %s", containerCfg.Image)
	}
	if string(hostCfg.NetworkMode) != "default" {
		t.Fatalf("unexpected network mode: %s", hostCfg.NetworkMode)
	}
	if !strings.Contains(strings.Join(containerCfg.Cmd, " "), "--snap-time=90") {
		t.Fatalf("expected snap-time override in cmd: %v", containerCfg.Cmd)
	}
	if scriptContent != defaults.PackScript {
		t.Fatalf("expected pack script content")
	}
	if scriptPath != filepath.Join("/tmp/docker", "~zod", "_data", "start_urbit.sh") {
		t.Fatalf("unexpected script path: %s", scriptPath)
	}
	if shipState["~zod"].BootStatus != "noboot" {
		t.Fatalf("expected boot status reset to noboot, got %s", shipState["~zod"].BootStatus)
	}
}

func TestUrbitContainerConfWireguardAndCustomPier(t *testing.T) {
	rt := testUrbitRuntime()
	rt.runtimeSettings = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{SnapTime: 60} }
	shipState := map[string]structs.UrbitDocker{
		"~nec": {
			PierName:           "~nec",
			BootStatus:         "boot",
			Network:            "wireguard",
			WgHTTPPort:         8443,
			WgAmesPort:         45555,
			LoomSize:           33,
			DevMode:            true,
			SnapTime:           120,
			CustomPierLocation: "/custom/pier",
		},
	}
	rt.urbitConf = func(name string) structs.UrbitDocker { return shipState[name] }
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.getLatestContainerInfo = func(kind string) (map[string]string, error) {
		return map[string]string{"repo": "repo/" + kind, "tag": "tag", "hash": "hash"}, nil
	}
	rt.architecture = func() string { return "arm64" }
	rt.updateUrbit = func(name string, mutate func(*structs.UrbitDocker) error) error {
		c := shipState[name]
		if err := mutate(&c); err != nil {
			return err
		}
		shipState[name] = c
		return nil
	}
	var scriptPath string
	rt.writeFile = func(path string, _ []byte, _ os.FileMode) error {
		scriptPath = path
		return nil
	}

	containerCfg, hostCfg, err := urbitContainerConfWithRuntime(rt, "~nec")
	if err != nil {
		t.Fatalf("urbitContainerConf failed: %v", err)
	}
	cmd := strings.Join(containerCfg.Cmd, " ")
	if !strings.Contains(cmd, "--http-port=8443") || !strings.Contains(cmd, "--port=45555") || !strings.Contains(cmd, "--snap-time=120") || !strings.Contains(cmd, "--devmode=True") {
		t.Fatalf("unexpected wireguard cmd: %v", containerCfg.Cmd)
	}
	if string(hostCfg.NetworkMode) != "container:wireguard" {
		t.Fatalf("unexpected network mode: %s", hostCfg.NetworkMode)
	}
	if len(hostCfg.Mounts) != 1 || hostCfg.Mounts[0].Source != "/custom/pier" || hostCfg.Mounts[0].Type != mount.TypeBind {
		t.Fatalf("unexpected mounts: %+v", hostCfg.Mounts)
	}
	if scriptPath != "/custom/pier/start_urbit.sh" {
		t.Fatalf("unexpected custom script path: %s", scriptPath)
	}
}

func TestUrbitContainerConfErrors(t *testing.T) {
	rt := testUrbitRuntime()
	rt.runtimeSettings = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{} }
	rt.urbitConf = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{PierName: "~zod", BootStatus: "unknown"}
	}
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.getLatestContainerInfo = func(kind string) (map[string]string, error) {
		if kind == "minio" {
			return nil, errors.New("minio lookup failed")
		}
		return map[string]string{"repo": "repo/vere", "tag": "tag", "hash": "hash"}, nil
	}

	if _, _, err := urbitContainerConfWithRuntime(rt, "~zod"); err == nil || !strings.Contains(err.Error(), "minio lookup failed") {
		t.Fatalf("expected minio lookup error, got %v", err)
	}

	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/x", "tag": "tag", "hash": "hash"}, nil
	}
	if _, _, err := urbitContainerConfWithRuntime(rt, "~zod"); err == nil || !strings.Contains(err.Error(), "Unknown action") {
		t.Fatalf("expected unknown action error, got %v", err)
	}
}
