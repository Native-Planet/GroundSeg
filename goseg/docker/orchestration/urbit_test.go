package orchestration

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

func testUrbitRuntime() UrbitRuntime {
	return UrbitRuntime{
		RuntimeSnapshotOps: RuntimeSnapshotOps{
			ShipSettingsSnapshotFn:     func() config.ShipSettings { return config.ShipSettings{} },
			ShipRuntimeSettingsSnapshotFn: func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{} },
		},
		RuntimeUrbitOps: RuntimeUrbitOps{
			LoadUrbitConfigFn: func(string) error { return nil },
			UrbitConfFn: func(string) structs.UrbitDocker { return structs.UrbitDocker{} },
			UpdateUrbitFn: func(string, func(*structs.UrbitDocker) error) error {
				return nil
			},
		},
		RuntimeContainerOps: RuntimeContainerOps{
			StartContainerFn:      func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
			CreateContainerFn:     func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
			UpdateContainerStateFn: func(string, structs.ContainerState) {},
		},
		RuntimeImageOps: RuntimeImageOps{
			GetLatestContainerInfoFn: func(string) (map[string]string, error) {
				return map[string]string{"repo": "repo", "tag": "tag", "hash": "hash"}, nil
			},
		},
		RuntimeContextOps: RuntimeContextOps{
			ArchitectureFn: func() string { return "amd64" },
			DockerDirFn:    func() string { return "/tmp/docker" },
		},
		RuntimeFileOps: RuntimeFileOps{
			WriteFileFn: func(string, []byte, os.FileMode) error { return nil },
		},
	}
}

func TestLoadUrbitsDispatchesStartOrCreate(t *testing.T) {
	rt := testUrbitRuntime()
	rt.ShipSettingsSnapshotFn = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"~zod", "~bus", "~mar"}}
	}
	rt.LoadUrbitConfigFn = func(pier string) error {
		if pier == "~bus" {
			return errors.New("bad config")
		}
		return nil
	}
	rt.UrbitConfFn = func(pier string) structs.UrbitDocker {
		if pier == "~mar" {
			return structs.UrbitDocker{
				UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
					BootStatus: "noboot",
				},
			}
		}
		return structs.UrbitDocker{
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				BootStatus: "boot",
			},
		}
	}
	var started, created, updated []string
	rt.StartContainerFn = func(name, ctype string) (structs.ContainerState, error) {
		started = append(started, name+":"+ctype)
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	rt.CreateContainerFn = func(name, ctype string) (structs.ContainerState, error) {
		created = append(created, name+":"+ctype)
		return structs.ContainerState{ActualStatus: "created"}, nil
	}
	rt.UpdateContainerStateFn = func(name string, _ structs.ContainerState) {
		updated = append(updated, name)
	}

	if err := loadUrbits(rt); err == nil {
		t.Fatalf("expected LoadUrbits failure due to partial bootstrap errors")
	} else if !strings.Contains(err.Error(), "bad config") {
		t.Fatalf("unexpected LoadUrbits error: %v", err)
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

func TestLoadUrbitsReturnsErrorWhenAllShipsFail(t *testing.T) {
	rt := testUrbitRuntime()
	rt.ShipSettingsSnapshotFn = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"~zod", "~bus"}}
	}
	rt.LoadUrbitConfigFn = func(string) error { return errors.New("config missing") }

	loadErr := loadUrbits(rt)
	if loadErr == nil {
		t.Fatal("expected LoadUrbits to fail when all configured piers fail")
	}
	if !strings.Contains(loadErr.Error(), "load urbits failed for one or more ships") {
		t.Fatalf("expected aggregated urbit failure summary, got: %v", loadErr)
	}
}

func TestUrbitContainerConfDefaultNetworkAndPackReset(t *testing.T) {
	shipState := map[string]structs.UrbitDocker{
		"~zod": {
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				PierName:         "~zod",
				BootStatus:       "pack",
				HTTPPort:         8080,
				AmesPort:         34344,
				LoomSize:         31,
				UrbitAmd64Sha256: "old-urbit",
				MinioAmd64Sha256: "old-minio",
			},
		},
	}
	rt := testUrbitRuntime()
	rt.ShipRuntimeSettingsSnapshotFn = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{SnapTime: 90} }
	rt.LoadUrbitConfigFn = func(string) error { return nil }
	rt.UpdateUrbitFn = func(name string, mutate func(*structs.UrbitDocker) error) error {
		c := shipState[name]
		if err := mutate(&c); err != nil {
			return err
		}
		shipState[name] = c
		return nil
	}
	rt.UrbitConfFn = func(name string) structs.UrbitDocker { return shipState[name] }
	rt.GetLatestContainerInfoFn = func(kind string) (map[string]string, error) {
		switch kind {
		case "vere":
			return map[string]string{"repo": "repo/vere", "tag": "v4.0", "hash": "new-urbit"}, nil
		case "minio":
			return map[string]string{"repo": "repo/minio", "tag": "latest", "hash": "new-minio"}, nil
		default:
			return nil, errors.New("unknown kind")
		}
	}
	var scriptPath, scriptContent string
	rt.WriteFileFn = func(path string, data []byte, _ os.FileMode) error {
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

func TestUrbitContainerConfReloadsConfigAfterMutation(t *testing.T) {
	shipState := map[string]structs.UrbitDocker{
		"~zod": {
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				PierName:         "~zod",
				BootStatus:       "boot",
				HTTPPort:         8080,
				AmesPort:         34344,
				LoomSize:         31,
				UrbitAmd64Sha256: "old-urbit",
				MinioAmd64Sha256: "old-minio",
			},
		},
	}

	rt := testUrbitRuntime()
	rt.ShipRuntimeSettingsSnapshotFn = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{} }
	rt.LoadUrbitConfigFn = func(string) error { return nil }
	rt.UpdateUrbitFn = func(name string, mutate func(*structs.UrbitDocker) error) error {
		c := shipState[name]
		if err := mutate(&c); err != nil {
			return err
		}
		c.BootStatus = "pack"
		shipState[name] = c
		return nil
	}
	rt.UrbitConfFn = func(name string) structs.UrbitDocker {
		return shipState[name]
	}
	rt.GetLatestContainerInfoFn = func(kind string) (map[string]string, error) {
		if kind == "minio" {
			return map[string]string{"repo": "repo/minio", "tag": "tag", "hash": "new-minio"}, nil
		}
		return map[string]string{"repo": "repo/vere", "tag": "tag", "hash": "new-urbit"}, nil
	}

	var scriptContent string
	rt.WriteFileFn = func(_ string, data []byte, _ os.FileMode) error {
		scriptContent = string(data)
		return nil
	}

	_, _, err := urbitContainerConfWithRuntime(rt, "~zod")
	if err != nil {
		t.Fatalf("urbitContainerConf failed: %v", err)
	}
	if scriptContent != defaults.PackScript {
		t.Fatalf("expected pack script content after persisted config update, got %q", scriptContent)
	}
}

func TestUrbitContainerConfWireguardAndCustomPier(t *testing.T) {
	shipState := map[string]structs.UrbitDocker{
		"~nec": {
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				PierName:   "~nec",
				BootStatus: "boot",
				LoomSize:   33,
				SnapTime:   120,
			},
			UrbitNetworkConfig: structs.UrbitNetworkConfig{
				Network:    "wireguard",
				WgHTTPPort: 8443,
				WgAmesPort: 45555,
			},
			UrbitFeatureConfig: structs.UrbitFeatureConfig{
				DevMode: true,
			},
			UrbitWebConfig: structs.UrbitWebConfig{
				CustomPierLocation: "/custom/pier",
			},
		},
	}
	rt := testUrbitRuntime()
	rt.ShipRuntimeSettingsSnapshotFn = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{SnapTime: 60} }
	rt.LoadUrbitConfigFn = func(pier string) error { return nil }
	rt.UpdateUrbitFn = func(name string, mutate func(*structs.UrbitDocker) error) error {
		c := shipState[name]
		if err := mutate(&c); err != nil {
			return err
		}
		shipState[name] = c
		return nil
	}
	rt.UrbitConfFn = func(name string) structs.UrbitDocker { return shipState[name] }
	rt.GetLatestContainerInfoFn = func(kind string) (map[string]string, error) {
		return map[string]string{"repo": "repo/" + kind, "tag": "tag", "hash": "hash"}, nil
	}
	rt.ArchitectureFn = func() string { return "arm64" }
	rt.DockerDirFn = func() string { return "/custom/pier" }
	var scriptPath string
	rt.WriteFileFn = func(path string, _ []byte, _ os.FileMode) error {
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
	rt.LoadUrbitConfigFn = func(string) error { return nil }
	rt.UrbitConfFn = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				PierName:   "~zod",
				BootStatus: "unknown",
			},
		}
	}
	rt.UpdateUrbitFn = func(string, func(*structs.UrbitDocker) error) error { return nil }
	rt.ShipRuntimeSettingsSnapshotFn = func() config.ShipRuntimeSettings { return config.ShipRuntimeSettings{} }
	rt.GetLatestContainerInfoFn = func(kind string) (map[string]string, error) {
		if kind == "minio" {
			return nil, errors.New("minio lookup failed")
		}
		return map[string]string{"repo": "repo/vere", "tag": "tag", "hash": "hash"}, nil
	}

	if _, _, err := urbitContainerConfWithRuntime(rt, "~zod"); err == nil || !strings.Contains(err.Error(), "minio lookup failed") {
		t.Fatalf("expected minio lookup error, got %v", err)
	}

	rt.GetLatestContainerInfoFn = func(kind string) (map[string]string, error) {
		return map[string]string{"repo": "repo/x", "tag": "tag", "hash": "hash"}, nil
	}
	if _, _, err := urbitContainerConfWithRuntime(rt, "~zod"); err == nil || !strings.Contains(err.Error(), "Unknown action") {
		t.Fatalf("expected unknown action error, got %v", err)
	}
}
