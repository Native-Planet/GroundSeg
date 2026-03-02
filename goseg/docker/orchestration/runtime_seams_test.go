package orchestration

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
)

type runtimeContainerTestOps struct {
	startContainerFn       func(string, string) (structs.ContainerState, error)
	stopContainerByNameFn  func(string) error
	restartContainerFn     func(string) error
	deleteContainerFn      func(string) error
	getContainerStateFn    func() map[string]structs.ContainerState
	updateContainerStateFn func(string, structs.ContainerState)
	getShipStatusFn        func([]string) (map[string]string, error)
	waitForShipExitFn      func(string, time.Duration) error
}

func (ops runtimeContainerTestOps) runtimeOps() RuntimeContainerOps {
	return RuntimeContainerOps{
		StartContainerFn:      ops.startContainerFn,
		StopContainerByNameFn: ops.stopContainerByNameFn,
		RestartContainerFn:    ops.restartContainerFn,
		DeleteContainerFn:     ops.deleteContainerFn,
		GetContainerStateFn:   ops.getContainerStateFn,
		UpdateContainerStateFn: func(name string, state structs.ContainerState) {
			if ops.updateContainerStateFn != nil {
				ops.updateContainerStateFn(name, state)
			}
		},
		GetShipStatusFn:   ops.getShipStatusFn,
		WaitForShipExitFn: ops.waitForShipExitFn,
	}
}

type runtimeUrbitTestOps struct {
	loadUrbitConfigFn     func(string) error
	urbitConfFn           func(string) structs.UrbitDocker
	updateUrbitFn         func(string, func(*structs.UrbitDocker) error) error
	getContainerNetworkFn func(string) (string, error)
	getLusCodeFn          func(string) (string, error)
	clearLusCodeFn        func(string)
}

func (ops runtimeUrbitTestOps) runtimeOps() RuntimeUrbitOps {
	return RuntimeUrbitOps{
		LoadUrbitConfigFn: ops.loadUrbitConfigFn,
		UrbitConfFn: func(patp string) structs.UrbitDocker {
			if ops.urbitConfFn == nil {
				return structs.UrbitDocker{}
			}
			return ops.urbitConfFn(patp)
		},
		UpdateUrbitFn: ops.updateUrbitFn,
		GetContainerNetworkFn: func(patp string) (string, error) {
			if ops.getContainerNetworkFn == nil {
				return "", nil
			}
			return ops.getContainerNetworkFn(patp)
		},
		GetLusCodeFn: func(patp string) (string, error) {
			if ops.getLusCodeFn == nil {
				return "", nil
			}
			return ops.getLusCodeFn(patp)
		},
		ClearLusCodeFn: func(patp string) {
			if ops.clearLusCodeFn != nil {
				ops.clearLusCodeFn(patp)
			}
		},
	}
}

type runtimeSnapshotTestOps struct {
	startramSettingsSnapshotFn func() config.StartramSettings
	shipSettingsSnapshotFn     func() config.ShipSettings
	getStartramConfigFn        func() structs.StartramRetrieve
	check502SettingsSnapshotFn func() config.Check502Settings
}

func (ops runtimeSnapshotTestOps) runtimeOps() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: func() config.StartramSettings {
			if ops.startramSettingsSnapshotFn == nil {
				return config.StartramSettings{}
			}
			return ops.startramSettingsSnapshotFn()
		},
		ShipSettingsSnapshotFn: func() config.ShipSettings {
			if ops.shipSettingsSnapshotFn == nil {
				return config.ShipSettings{}
			}
			return ops.shipSettingsSnapshotFn()
		},
		GetStartramConfigFn: func() structs.StartramRetrieve {
			if ops.getStartramConfigFn == nil {
				return structs.StartramRetrieve{}
			}
			return ops.getStartramConfigFn()
		},
		Check502SettingsSnapshotFn: func() config.Check502Settings {
			if ops.check502SettingsSnapshotFn == nil {
				return config.Check502Settings{}
			}
			return ops.check502SettingsSnapshotFn()
		},
	}
}

type runtimeConfigTestOps struct {
	updateConfTypedFn func(...config.ConfUpdateOption) error
	withWgOnFn        func(bool) config.ConfUpdateOption
	cycleWgKeyFn      func() error
	barExitFn         func(string) error
}

func (ops runtimeConfigTestOps) runtimeOps() RuntimeConfigOps {
	return RuntimeConfigOps{
		UpdateConfTypedFn: func(opts ...config.ConfUpdateOption) error {
			if ops.updateConfTypedFn == nil {
				return nil
			}
			return ops.updateConfTypedFn(opts...)
		},
		WithWgOnFn: func(enabled bool) config.ConfUpdateOption {
			if ops.withWgOnFn == nil {
				return config.WithWgOn(enabled)
			}
			return ops.withWgOnFn(enabled)
		},
		CycleWgKeyFn: func() error {
			if ops.cycleWgKeyFn == nil {
				return nil
			}
			return ops.cycleWgKeyFn()
		},
		BarExitFn: func(patp string) error {
			if ops.barExitFn == nil {
				return nil
			}
			return ops.barExitFn(patp)
		},
	}
}

type runtimeLoadTestOps struct {
	loadWireguardFn func() error
	loadMCFn        func() error
	loadMinIOsFn    func() error
	loadUrbitsFn    func() error
}

func (ops runtimeLoadTestOps) runtimeOps() RuntimeLoadOps {
	return RuntimeLoadOps{
		LoadWireguardFn: func() error {
			if ops.loadWireguardFn == nil {
				return nil
			}
			return ops.loadWireguardFn()
		},
		LoadMCFn: func() error {
			if ops.loadMCFn == nil {
				return nil
			}
			return ops.loadMCFn()
		},
		LoadMinIOsFn: func() error {
			if ops.loadMinIOsFn == nil {
				return nil
			}
			return ops.loadMinIOsFn()
		},
		LoadUrbitsFn: func() error {
			if ops.loadUrbitsFn == nil {
				return nil
			}
			return ops.loadUrbitsFn()
		},
	}
}

type runtimeServiceTestOps struct {
	svcDeleteFn func(string, string) error
}

func (ops runtimeServiceTestOps) runtimeOps() RuntimeServiceOps {
	return RuntimeServiceOps{
		SvcDeleteFn: func(patp, kind string) error {
			if ops.svcDeleteFn == nil {
				return nil
			}
			return ops.svcDeleteFn(patp, kind)
		},
	}
}

type startupBootstrapTestOps struct {
	initializeFn func() error
}

func (ops startupBootstrapTestOps) runtimeOps() StartupBootstrapOps {
	return StartupBootstrapOps{
		Initialize: func() error {
			if ops.initializeFn == nil {
				return nil
			}
			return ops.initializeFn()
		},
	}
}

type startupImageTestOps struct {
	getLatestContainerInfoFn func(string) (map[string]string, error)
	pullImageIfNotExistFn    func(string, map[string]string) (bool, error)
}

func (ops startupImageTestOps) runtimeOps() StartupImageOps {
	return StartupImageOps{
		GetLatestContainerInfo: ops.getLatestContainerInfoFn,
		PullImageIfNotExist:    ops.pullImageIfNotExistFn,
	}
}

type startupLoadTestOps struct {
	loadWireguardFn func() error
	loadMCFn        func() error
	loadMinIOsFn    func() error
	loadNetdataFn   func() error
	loadUrbitsFn    func() error
	loadLlamaFn     func() error
}

func (ops startupLoadTestOps) runtimeOps() StartupLoadOps {
	return StartupLoadOps{
		LoadWireguard: func() error {
			if ops.loadWireguardFn == nil {
				return nil
			}
			return ops.loadWireguardFn()
		},
		LoadMC: func() error {
			if ops.loadMCFn == nil {
				return nil
			}
			return ops.loadMCFn()
		},
		LoadMinIOs: func() error {
			if ops.loadMinIOsFn == nil {
				return nil
			}
			return ops.loadMinIOsFn()
		},
		LoadNetdata: func() error {
			if ops.loadNetdataFn == nil {
				return nil
			}
			return ops.loadNetdataFn()
		},
		LoadUrbits: func() error {
			if ops.loadUrbitsFn == nil {
				return nil
			}
			return ops.loadUrbitsFn()
		},
		LoadLlama: func() error {
			if ops.loadLlamaFn == nil {
				return nil
			}
			return ops.loadLlamaFn()
		},
	}
}

func TestNewRuntimeDelegatesToOverrideFns(t *testing.T) {
	var (
		startCalled               bool
		stopCalled                bool
		restartCalled             bool
		deleteCalled              bool
		getStateCalled            bool
		updateStateCalled         bool
		getStatusCalled           bool
		waitCalled                bool
		loadUrbitConfigCalled     bool
		urbitConfCalled           bool
		updateUrbitCalled         bool
		getContainerNetworkCalled bool
		getLusCodeCalled          bool
		clearLusCodeCalled        bool
		startramSnapshotCalled    bool
		shipSnapshotCalled        bool
		getStartramConfigCalled   bool
		check502SnapshotCalled    bool
		updateConfCalled          bool
		withWgOnCalled            bool
		cycleWgKeyCalled          bool
		barExitCalled             bool
		loadWireguardCalled       bool
		loadMCCalled              bool
		loadMinIOsCalled          bool
		loadUrbitsCalled          bool
		svcDeleteCalled           bool
	)

	rt := NewRuntime(
		WithContainerOps(runtimeContainerTestOps{
			startContainerFn: func(string, string) (structs.ContainerState, error) {
				startCalled = true
				return structs.ContainerState{ActualStatus: "running"}, nil
			},
			stopContainerByNameFn: func(string) error {
				stopCalled = true
				return nil
			},
			restartContainerFn: func(string) error {
				restartCalled = true
				return nil
			},
			deleteContainerFn: func(string) error {
				deleteCalled = true
				return nil
			},
			getContainerStateFn: func() map[string]structs.ContainerState {
				getStateCalled = true
				return map[string]structs.ContainerState{"wireguard": {ActualStatus: "running"}}
			},
			updateContainerStateFn: func(string, structs.ContainerState) {
				updateStateCalled = true
			},
			getShipStatusFn: func([]string) (map[string]string, error) {
				getStatusCalled = true
				return map[string]string{"~zod": "Up"}, nil
			},
			waitForShipExitFn: func(string, time.Duration) error {
				waitCalled = true
				return nil
			},
		}.runtimeOps()),
		WithUrbitOps(runtimeUrbitTestOps{
			loadUrbitConfigFn: func(string) error {
				loadUrbitConfigCalled = true
				return nil
			},
			urbitConfFn: func(string) structs.UrbitDocker {
				urbitConfCalled = true
				return structs.UrbitDocker{}
			},
			updateUrbitFn: func(string, func(*structs.UrbitDocker) error) error {
				updateUrbitCalled = true
				return nil
			},
			getContainerNetworkFn: func(string) (string, error) {
				getContainerNetworkCalled = true
				return "wireguard", nil
			},
			getLusCodeFn: func(string) (string, error) {
				getLusCodeCalled = true
				return "lus", nil
			},
			clearLusCodeFn: func(string) {
				clearLusCodeCalled = true
			},
		}.runtimeOps()),
		WithSnapshotOps(runtimeSnapshotTestOps{
			startramSettingsSnapshotFn: func() config.StartramSettings {
				startramSnapshotCalled = true
				return config.StartramSettings{}
			},
			shipSettingsSnapshotFn: func() config.ShipSettings {
				shipSnapshotCalled = true
				return config.ShipSettings{}
			},
			getStartramConfigFn: func() structs.StartramRetrieve {
				getStartramConfigCalled = true
				return structs.StartramRetrieve{}
			},
			check502SettingsSnapshotFn: func() config.Check502Settings {
				check502SnapshotCalled = true
				return config.Check502Settings{}
			},
		}.runtimeOps()),
		WithConfigOps(runtimeConfigTestOps{
			updateConfTypedFn: func(...config.ConfUpdateOption) error {
				updateConfCalled = true
				return nil
			},
			withWgOnFn: func(enabled bool) config.ConfUpdateOption {
				withWgOnCalled = true
				return config.WithWgOn(enabled)
			},
			cycleWgKeyFn: func() error {
				cycleWgKeyCalled = true
				return nil
			},
			barExitFn: func(string) error {
				barExitCalled = true
				return nil
			},
		}.runtimeOps()),
		WithLoadOps(runtimeLoadTestOps{
			loadWireguardFn: func() error {
				loadWireguardCalled = true
				return nil
			},
			loadMCFn: func() error {
				loadMCCalled = true
				return nil
			},
			loadMinIOsFn: func() error {
				loadMinIOsCalled = true
				return nil
			},
			loadUrbitsFn: func() error {
				loadUrbitsCalled = true
				return nil
			},
		}.runtimeOps()),
		WithServiceOps(runtimeServiceTestOps{
			svcDeleteFn: func(string, string) error {
				svcDeleteCalled = true
				return nil
			},
		}.runtimeOps()),
	)

	if _, err := rt.StartContainerFn("wireguard", "wireguard"); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := rt.StopContainerByNameFn("wireguard"); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	if err := rt.RestartContainerFn("wireguard"); err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if err := rt.DeleteContainerFn("wireguard"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if got := rt.GetContainerStateFn(); len(got) != 1 || got["wireguard"].ActualStatus != "running" {
		t.Fatalf("unexpected state map: %+v", got)
	}
	rt.UpdateContainerStateFn("wireguard", structs.ContainerState{ActualStatus: "running"})
	if _, err := rt.GetShipStatusFn([]string{"~zod"}); err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if err := rt.WaitForShipExitFn("~zod", time.Second); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if err := rt.LoadUrbitConfigFn("~zod"); err != nil {
		t.Fatalf("unexpected urbit config load error: %v", err)
	}
	if got := rt.UrbitConfFn("~zod"); !reflect.DeepEqual(got, structs.UrbitDocker{}) {
		t.Fatalf("unexpected urbit conf: %+v", got)
	}
	if err := rt.UpdateUrbitFn("~zod", func(conf *structs.UrbitDocker) error { return nil }); err != nil {
		t.Fatalf("unexpected update urbit error: %v", err)
	}
	if got, err := rt.GetContainerNetworkFn("~zod"); err != nil || got != "wireguard" {
		t.Fatalf("unexpected network lookup: %s %v", got, err)
	}
	if got, err := rt.GetLusCodeFn("~zod"); err != nil || got != "lus" {
		t.Fatalf("unexpected lus lookup: %s %v", got, err)
	}
	rt.ClearLusCodeFn("~zod")
	if got := rt.StartramSettingsSnapshotFn(); !reflect.DeepEqual(got, config.StartramSettings{}) {
		t.Fatalf("unexpected startram snapshot: %+v", got)
	}
	if got := rt.ShipSettingsSnapshotFn(); !reflect.DeepEqual(got, config.ShipSettings{}) {
		t.Fatalf("unexpected ship snapshot: %+v", got)
	}
	if got := rt.GetStartramConfigFn(); !reflect.DeepEqual(got, structs.StartramRetrieve{}) {
		t.Fatalf("unexpected startram config: %+v", got)
	}
	if got := rt.Check502SettingsSnapshotFn(); !reflect.DeepEqual(got, config.Check502Settings{}) {
		t.Fatalf("unexpected check502 snapshot: %+v", got)
	}
	if err := rt.UpdateConfTypedFn(config.WithWgOn(true)); err != nil {
		t.Fatalf("unexpected update conf error: %v", err)
	}
	if got := rt.WithWgOnFn(true); got == nil {
		t.Fatalf("expected WithWgOnFn to return a config option")
	}
	if err := rt.CycleWgKeyFn(); err != nil {
		t.Fatalf("unexpected cycle wg key error: %v", err)
	}
	if err := rt.BarExitFn("~zod"); err != nil {
		t.Fatalf("unexpected bar exit error: %v", err)
	}
	if err := rt.LoadWireguardFn(); err != nil {
		t.Fatalf("unexpected load wireguard error: %v", err)
	}
	if err := rt.LoadMCFn(); err != nil {
		t.Fatalf("unexpected load mc error: %v", err)
	}
	if err := rt.LoadMinIOsFn(); err != nil {
		t.Fatalf("unexpected load minios error: %v", err)
	}
	if err := rt.LoadUrbitsFn(); err != nil {
		t.Fatalf("unexpected load urbits error: %v", err)
	}
	if err := rt.SvcDeleteFn("~zod", "mc"); err != nil {
		t.Fatalf("unexpected svc delete error: %v", err)
	}

	assert := func(cond bool, name string) {
		if !cond {
			t.Fatalf("expected callback to run: %s", name)
		}
	}
	assert(startCalled, "start")
	assert(stopCalled, "stop")
	assert(restartCalled, "restart")
	assert(deleteCalled, "delete")
	assert(getStateCalled, "get-state")
	assert(updateStateCalled, "update-state")
	assert(getStatusCalled, "get-status")
	assert(waitCalled, "wait")
	assert(loadUrbitConfigCalled, "load-urbit-conf")
	assert(urbitConfCalled, "urbit-conf")
	assert(updateUrbitCalled, "update-urbit")
	assert(getContainerNetworkCalled, "container-network")
	assert(getLusCodeCalled, "lus-code")
	assert(clearLusCodeCalled, "clear-lus")
	assert(startramSnapshotCalled, "startram-snapshot")
	assert(shipSnapshotCalled, "ship-snapshot")
	assert(getStartramConfigCalled, "startram-config")
	assert(check502SnapshotCalled, "check502")
	assert(updateConfCalled, "update-conf")
	assert(withWgOnCalled, "with-wg")
	assert(cycleWgKeyCalled, "cycle-wg-key")
	assert(barExitCalled, "bar-exit")
	assert(loadWireguardCalled, "load-wireguard")
	assert(loadMCCalled, "load-mc")
	assert(loadMinIOsCalled, "load-minios")
	assert(loadUrbitsCalled, "load-urbits")
	assert(svcDeleteCalled, "svc-delete")
}

func TestNewStartupRuntimeRespectsOverrideFns(t *testing.T) {
	bootstrapCalled := false
	bootstrapErr := errors.New("bootstrap failed")
	pullImageErr := errors.New("pull failed")
	imageContainerType := ""
	imageContainerTypePulled := ""
	load := struct {
		wireguardCalled bool
		mcCalled        bool
		minioCalled     bool
		urbitsCalled    bool
		netdataCalled   bool
		llamaCalled     bool
	}{}

	runtime := NewStartupRuntime(
		WithStartupBootstrapOps(startupBootstrapTestOps{
			initializeFn: func() error {
				bootstrapCalled = true
				return bootstrapErr
			},
		}.runtimeOps()),
		WithStartupImageOps(startupImageTestOps{
			getLatestContainerInfoFn: func(kind string) (map[string]string, error) {
				imageContainerType = kind
				return map[string]string{"kind": kind}, nil
			},
			pullImageIfNotExistFn: func(kind string, imageInfo map[string]string) (bool, error) {
				imageContainerTypePulled = kind
				if kind != "netdata" || len(imageInfo) == 0 {
					return true, pullImageErr
				}
				return true, nil
			},
		}.runtimeOps()),
		WithStartupLoadOps(startupLoadTestOps{
			loadWireguardFn: func() error {
				load.wireguardCalled = true
				return nil
			},
			loadMCFn: func() error {
				load.mcCalled = true
				return nil
			},
			loadMinIOsFn: func() error {
				load.minioCalled = true
				return nil
			},
			loadNetdataFn: func() error {
				load.netdataCalled = true
				return nil
			},
			loadUrbitsFn: func() error {
				load.urbitsCalled = true
				return nil
			},
			loadLlamaFn: func() error {
				load.llamaCalled = true
				return nil
			},
		}.runtimeOps()),
	)
	if err := runtime.Initialize(); err == nil || !strings.Contains(err.Error(), "bootstrap failed") {
		t.Fatalf("expected bootstrap override to execute and return error")
	}
	if !bootstrapCalled {
		t.Fatalf("expected bootstrap override to be called")
	}
	if _, err := runtime.GetLatestContainerInfo("netdata"); err != nil {
		t.Fatalf("unexpected image info error: %v", err)
	}
	if imageContainerType != "netdata" {
		t.Fatalf("expected image type netdata, got %s", imageContainerType)
	}
	if _, err := runtime.PullImageIfNotExist("netdata", map[string]string{"tag": "latest"}); err != nil {
		t.Fatalf("expected pull image override to return nil")
	}
	if imageContainerTypePulled != "netdata" {
		t.Fatalf("expected pull type netdata, got %s", imageContainerTypePulled)
	}
	if load.urbitsCalled || load.wireguardCalled || load.mcCalled || load.minioCalled || load.netdataCalled || load.llamaCalled {
		t.Fatalf("expected startup load overrides to not run before invoke")
	}
	if err := runtime.LoadWireguard(); err != nil {
		t.Fatalf("expected overridden load wireguard to succeed")
	}
	if err := runtime.LoadMC(); err != nil {
		t.Fatalf("expected overridden load mc to succeed")
	}
	if err := runtime.LoadMinIOs(); err != nil {
		t.Fatalf("expected overridden load minios to succeed")
	}
	if err := runtime.LoadLlama(); err != nil {
		t.Fatalf("expected overridden load llama to succeed")
	}
	if err := runtime.LoadUrbits(); err != nil {
		t.Fatalf("expected overridden load urbits to succeed")
	}
	if err := runtime.LoadNetdata(); err != nil {
		t.Fatalf("expected overridden load netdata to succeed")
	}
	if !load.urbitsCalled || !load.wireguardCalled || !load.mcCalled || !load.minioCalled || !load.netdataCalled || !load.llamaCalled {
		t.Fatalf("expected startup load overrides to run")
	}
}

func TestNewStartramRuntimeUsesOverridesForServiceLoaders(t *testing.T) {
	var servicesCalled bool
	var regionsCalled bool
	runtime := NewStartramRuntime(WithStartramServiceLoaders(
		func() error {
			servicesCalled = true
			return nil
		},
		func() error {
			regionsCalled = true
			return errors.New("regions failed")
		},
	))
	if err := runtime.GetStartramServicesFn(); err != nil {
		t.Fatalf("unexpected service loader error: %v", err)
	}
	if err := runtime.LoadStartramRegionsFn(); err == nil || !strings.Contains(err.Error(), "regions failed") {
		t.Fatalf("expected service region error, got %v", err)
	}
	if !servicesCalled || !regionsCalled {
		t.Fatalf("expected both service loader callbacks to run")
	}
}

func TestMinioRuntimeFromDockerWiresRuntimeDependencies(t *testing.T) {
	rt := newDockerRuntime()
	rt.contextOps = RuntimeContextOps{
		BasePathFn:  func() string { return "/tmp/base" },
		DockerDirFn: func() string { return "/tmp/docker" },
	}
	rt.configOps = RuntimeSnapshotOps{
		ConfFn:                    func() structs.SysConfig { return structs.SysConfig{} },
		ShipSettingsSnapshotFn:    func() config.ShipSettings { return config.ShipSettings{} },
	}
	rt.fileOps = RuntimeFileOps{
		OpenFn:      func(string) (*os.File, error) { return nil, nil },
		ReadFileFn:  func(string) ([]byte, error) { return nil, nil },
		WriteFileFn: func(string, []byte, os.FileMode) error { return nil },
		MkdirAllFn:  func(string, os.FileMode) error { return nil },
	}
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (map[string]string, error) {
			return map[string]string{"repo": "repo", "tag": "tag", "hash": "hash"}, nil
		},
		GetLatestContainerImageFn: func(string) (string, error) {
			return "image", nil
		},
	}
	rt.containerOps = RuntimeContainerOps{
		StartContainerFn:            func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
		UpdateContainerStateFn:      func(string, structs.ContainerState) {},
		GetContainerRunningStatusFn: func(string) (string, error) { return "Up", nil },
		CreateContainerFn:           func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
	}
	rt.commandOps = RuntimeCommandOps{
		CopyFileToVolumeFn: func(string, string, string, string, volumeWriterImageSelector) error { return nil },
	}
	rt.volumeOps = RuntimeVolumeOps{
		VolumeExistsFn: func(string) (bool, error) { return true, nil },
		CreateVolumeFn: func(string) error { return nil },
	}
	rt.timerOps = RuntimeTimerOps{
		SleepFn:        func(time.Duration) {},
		PollIntervalFn: func() time.Duration { return 100 * time.Millisecond },
	}
	rt.commandOps.RandReadFn = func([]byte) (int, error) { return 0, nil }
	rt.urbitOps = RuntimeUrbitOps{
		LoadUrbitConfigFn: func(string) error { return nil },
		UrbitConfFn:       func(string) structs.UrbitDocker { return structs.UrbitDocker{} },
		UpdateUrbitFn:     func(string, func(*structs.UrbitDocker) error) error { return nil },
	}
	rt.minioOps = RuntimeMinioOps{
		SetMinIOPasswordFn:    func(string, string) error { return nil },
		GetMinIOPasswordFn:    func(string) (string, error) { return "pw", nil },
		CreateDefaultMcConfFn: func() error { return nil },
	}
	rt.netdataOps = RuntimeNetdataOps{
		CreateDefaultNetdataConfFn: func() error { return nil },
		WriteNDConfFn:              nil,
	}

	minioRuntime := minioRuntimeFromDocker(rt)
	if err := minioRuntime.LoadUrbitConfigFn("ship"); err != nil {
		t.Fatalf("unexpected load urbit config error: %v", err)
	}
	if _, err := minioRuntime.GetLatestContainerInfoFn("minio"); err != nil {
		t.Fatalf("unexpected image metadata error: %v", err)
	}
	if err := minioRuntime.SetMinIOPasswordFn("minio~ship", "secret"); err != nil {
		t.Fatalf("unexpected set password error: %v", err)
	}
	if pwd, err := minioRuntime.GetMinIOPasswordFn("ship"); err != nil || pwd != "pw" {
		t.Fatalf("unexpected get minio password: %s %v", pwd, err)
	}
	status, err := minioRuntime.GetContainerRunningStatusFn("mc")
	if err != nil {
		t.Fatalf("unexpected container status error: %v", err)
	}
	if status != "Up" { // ensure copied function
		t.Fatal("expected running status seam to be wired")
	}
}
