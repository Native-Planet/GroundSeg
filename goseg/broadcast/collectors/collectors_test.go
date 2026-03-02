package collectors

import (
	"errors"
	"testing"
	"time"

	"groundseg/structs"
	"groundseg/transition"
)

func collectorRuntimeWith(overrides func(*collectorRuntime)) collectorRuntime {
	runtime := defaultCollectorRuntime()
	if overrides != nil {
		overrides(&runtime)
	}
	return runtime
}

func TestCollectUrbitDeploymentInputsBuildsDeploymentState(t *testing.T) {
	runtime := collectorRuntimeWith(func(runtime *collectorRuntime) {
		runtime.loadUrbitConfigFn = func(string) error { return nil }
		runtime.urbitConfFn = func(string) structs.UrbitDocker {
			return structs.UrbitDocker{
				HTTPPort:            8080,
				MinioPassword:       "default-minio",
				Network:             "bridge",
				WgURL:               "ship.wg.example",
				BootStatus:          "run",
				StartramReminder:    true,
				ChopOnUpgrade:       true,
				MeldDay:             "monday",
				MeldDate:            3,
				MeldFrequency:       7,
				ShowUrbitWeb:        "custom",
				CustomUrbitWeb:      "urbit.example.com",
				CustomS3Web:         "s3.example.com",
				DisableShipRestarts: true,
				RemoteTlonBackup:    true,
				LocalTlonBackup:     false,
				BackupTime:          "03:00",
				SnapTime:            2400,
			}
		}
		runtime.getMinIOPasswordFn = func(string) (string, error) {
			return "remote-minio", nil
		}
		runtime.getMinIOLinkedStatusFn = func(string) bool { return true }
	})

	inputs, ok := collectUrbitDeploymentInputsWithRuntime(
		runtime,
		"~zod",
		"host.internal",
		wireguardContext{
			registered: true,
			on:         true,
		},
		pierStartramSnapshot{
			remoteReadyByURL: map[string]bool{
				"ship.wg.example": true,
			},
		},
	)
	if !ok {
		t.Fatal("expected config resolution to succeed")
	}
	if inputs.url != "http://host.internal:8080" {
		t.Fatalf("unexpected url: %q", inputs.url)
	}
	if inputs.remote {
		t.Fatal("expected non-wireguard network")
	}
	if !inputs.remoteReady {
		t.Fatal("expected remoteReady lookup from snapshot map")
	}
	if inputs.minIOPwd != "remote-minio" {
		t.Fatalf("expected wireguard minio password, got %q", inputs.minIOPwd)
	}
	if !inputs.minIOLinked {
		t.Fatal("expected minio linked flag to be plumbed")
	}
}

func TestCollectUrbitRuntimeInputsCollectsRuntimeState(t *testing.T) {
	runtime := collectorRuntimeWith(func(runtime *collectorRuntime) {
		runtime.getContainerStatsFn = func(string) structs.ContainerStats {
			return structs.ContainerStats{
				MemoryUsage: 128,
				DiskUsage:   1024,
			}
		}
		runtime.lusCodeFn = func(string) (string, error) {
			return "0v2", nil
		}
		runtime.getDeskFn = func(_ string, desk string, _ bool) (string, error) {
			if desk == "penpai" {
				return string(transition.ContainerStatusRunning), nil
			}
			if desk == "groundseg" {
				return string(transition.ContainerStatusStopped), nil
			}
			return "missing", errors.New("unexpected desk")
		}
	})

	existing := structs.Urbit{}
	existing.Info.URL = "old-url"
	rtContext := urbitRuntimeContext{
		existingUrbits: map[string]structs.Urbit{
			"~zod": existing,
		},
		shipNetworks: map[string]string{
			"~zod": "bridge",
		},
	}

	scheduled := func(patp string) time.Time {
		if patp != "~zod" {
			return time.Time{}
		}
		t, _ := time.Parse(time.RFC3339, "2030-01-01T00:00:00Z")
		return t
	}

	inputs := collectUrbitRuntimeInputsWithRuntime(runtime, "~zod", "Up 1 minute", rtContext, scheduled)
	if inputs.network != "bridge" {
		t.Fatalf("expected network bridge, got %q", inputs.network)
	}
	if !inputs.isRunning {
		t.Fatal("expected running status for transition up status")
	}
	if inputs.lusCode != "0v2" {
		t.Fatalf("expected lus code 0v2, got %q", inputs.lusCode)
	}
	if inputs.lusCode == "" {
		t.Fatal("expected lus code to be populated")
	}
	if !inputs.penpaiCompanion || inputs.gallsegInstalled {
		t.Fatalf("unexpected desk install states: penpai=%v gallseg=%v", inputs.penpaiCompanion, inputs.gallsegInstalled)
	}
	if inputs.packUnixTime != 1893456000 {
		t.Fatalf("expected scheduled pack unix time 1893456000, got %d", inputs.packUnixTime)
	}
}

func TestComposeUrbitViewInputsCombinesRuntimeAndDeploymentInputs(t *testing.T) {
	runtimeInputs := urbitRuntimeInputs{
		existingUrbit: structs.Urbit{},
		dockerStats: structs.ContainerStats{
			MemoryUsage: 33,
			DiskUsage:   55,
		},
		network:          "bridge",
		isRunning:        true,
		lusCode:          "0v3",
		penpaiCompanion:  true,
		gallsegInstalled: true,
		packUnixTime:     10,
	}
	deploy := urbitDeploymentInputs{
		dockerConfig: structs.UrbitDocker{
			LoomSize:            4,
			DevMode:             true,
			UrbitVersion:        "v1.2.3",
			CustomUrbitWeb:      "urbit.example.com",
			CustomS3Web:         "s3.example.com",
			MeldSchedule:        true,
			MeldTime:            "11:00",
			MeldLast:            "2026-01-01",
			MeldScheduleType:    "weekly",
			MeldFrequency:       7,
			RemoteTlonBackup:    true,
			LocalTlonBackup:     false,
			DisableShipRestarts: true,
			SizeLimit:           2048,
			BackupTime:          "12:00",
			SnapTime:            3600,
		},
		url:                 "https://ship",
		remote:              true,
		remoteReady:         true,
		showUrbitWebAlias:   true,
		minIOPwd:            "secret",
		disableShipRestarts: true,
		minIOLinked:         true,
		startramReminder:    true,
		chopOnUpgrade:       false,
		packDay:             "Monday",
		packDate:            1,
		minIOURL:            "https://console.s3.ship",
		bootStatus:          true,
	}

	merged := composeUrbitViewInputs(runtimeInputs, deploy)
	if merged.network != "bridge" {
		t.Fatalf("expected network bridge, got %q", merged.network)
	}
	if merged.url != "https://ship" {
		t.Fatalf("unexpected url %q", merged.url)
	}
	if !merged.minIOLinked {
		t.Fatal("expected minio linked flag")
	}
	if merged.minIOPwd != "secret" {
		t.Fatalf("expected minio pwd secret, got %q", merged.minIOPwd)
	}
	if !merged.penpaiCompanion || !merged.gallsegInstalled {
		t.Fatalf("expected desk flags to be preserved")
	}
	if merged.packUnixTime != 10 {
		t.Fatalf("expected pack unix %d, got %d", 10, merged.packUnixTime)
	}
}

func TestCollectUrbitDeploymentInputsForPiersSkipsInvalidConfig(t *testing.T) {
	runtime := collectorRuntimeWith(func(runtime *collectorRuntime) {
		runtime.loadUrbitConfigFn = func(pier string) error {
			if pier == "~sam" {
				return errors.New("missing")
			}
			return nil
		}
		runtime.urbitConfFn = func(string) structs.UrbitDocker {
			return structs.UrbitDocker{WgURL: "ship.wg", Network: "bridge", HTTPPort: 8080}
		}
		runtime.getMinIOLinkedStatusFn = func(string) bool { return false }
	})

	inputs := collectUrbitDeploymentInputsForPiersWithRuntime(
		runtime,
		[]string{"~zod", "~sam"},
		"localhost",
		wireguardContext{},
		pierStartramSnapshot{remoteReadyByURL: map[string]bool{}},
	)
	if _, exists := inputs["~zod"]; !exists {
		t.Fatal("expected valid pier deployment inputs")
	}
	if _, exists := inputs["~sam"]; exists {
		t.Fatal("expected invalid pier to be skipped")
	}
}

func TestCollectUrbitRuntimeInputsForPiersCollectsAllShips(t *testing.T) {
	runtime := collectorRuntimeWith(func(runtime *collectorRuntime) {
		runtime.getContainerStatsFn = func(ship string) structs.ContainerStats {
			return structs.ContainerStats{MemoryUsage: uint64(len(ship))}
		}
		runtime.lusCodeFn = func(ship string) (string, error) {
			return ship + "-code", nil
		}
		runtime.getDeskFn = func(_ string, desk string, _ bool) (string, error) {
			if desk == "groundseg" || desk == "penpai" {
				return string(transition.ContainerStatusRunning), nil
			}
			return string(transition.ContainerStatusStopped), nil
		}
	})

	rtContext := urbitRuntimeContext{
		existingUrbits: map[string]structs.Urbit{},
		shipNetworks: map[string]string{
			"~zod": "net1",
			"~sam": "net2",
		},
	}
	inputs := collectUrbitRuntimeInputsForPiersWithRuntime(
		runtime,
		map[string]string{"~zod": "Up 1 second", "~sam": "Exited"},
		rtContext,
		func(patp string) time.Time {
			if patp == "~sam" {
				return time.Unix(123456, 0)
			}
			return time.Time{}
		},
	)
	if got := len(inputs); got != 2 {
		t.Fatalf("expected two runtime inputs, got %d", got)
	}
	zodInput, ok := inputs["~zod"]
	if !ok || !zodInput.isRunning {
		t.Fatalf("expected ~zod runtime input to be running with valid data: %+v", zodInput)
	}
	if zodInput.network != "net1" || zodInput.lusCode != "~zod-code" {
		t.Fatalf("unexpected zod runtime payload: %+v", zodInput)
	}
	if !zodInput.penpaiCompanion || !zodInput.gallsegInstalled {
		t.Fatalf("expected running desk flags to be true: %+v", zodInput)
	}
	samInput := inputs["~sam"]
	if samInput.isRunning {
		t.Fatal("expected stopped ship to not be running")
	}
	if samInput.network != "net2" {
		t.Fatal("expected net2")
	}
	if samInput.packUnixTime != 123456 {
		t.Fatalf("expected scheduled pack Unix time from callback, got %d", samInput.packUnixTime)
	}
}

func TestComposeUrbitViewsSkipsMissingDeploymentInput(t *testing.T) {
	runtimeInputs := map[string]urbitRuntimeInputs{
		"~zod": {
			existingUrbit: structs.Urbit{},
			isRunning:     true,
		},
		"~sam": {
			isRunning: true,
		},
	}
	deploymentInputs := map[string]urbitDeploymentInputs{
		"~sam": {
			dockerConfig: structs.UrbitDocker{
				LoomSize: 4,
			},
		},
	}
	backups := pierBackupSnapshot{
		remote:       structs.Backup{},
		localDaily:   structs.Backup{},
		localWeekly:  structs.Backup{},
		localMonthly: structs.Backup{},
	}
	updates := composeUrbitViews([]string{"~zod", "~sam"}, runtimeInputs, deploymentInputs, backups)
	if len(updates) != 1 {
		t.Fatalf("expected one composable urbit, got %d", len(updates))
	}
	if _, exists := updates["~sam"]; !exists {
		t.Fatal("expected sam to compose from valid deployment data")
	}
	if _, exists := updates["~zod"]; exists {
		t.Fatal("expected zod to be skipped due to missing deployment inputs")
	}
}
