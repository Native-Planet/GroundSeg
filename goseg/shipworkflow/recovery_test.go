package shipworkflow

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/structs"
)

type testRecoveryRuntimeContainer struct {
	getShipStatusFn    func([]string) (map[string]string, error)
	restartContainerFn func(string) error
	waitForShipExitFn  func(string, time.Duration) error
	deleteContainerFn  func(string) error
}

func (runtime testRecoveryRuntimeContainer) runtimeOps() orchestration.RuntimeContainerOps {
	return orchestration.RuntimeContainerOps{
		RuntimeContainerLifecycleOps: orchestration.RuntimeContainerLifecycleOps{
			StartContainerFn: func(string, string) (structs.ContainerState, error) {
				return structs.ContainerState{}, nil
			},
			StopContainerByNameFn: func(string) error { return nil },
			RestartContainerFn: func(patp string) error {
				if runtime.restartContainerFn == nil {
					return nil
				}
				return runtime.restartContainerFn(patp)
			},
			DeleteContainerFn: func(patp string) error {
				if runtime.deleteContainerFn == nil {
					return nil
				}
				return runtime.deleteContainerFn(patp)
			},
		},
		RuntimeContainerStateOps: orchestration.RuntimeContainerStateOps{
			GetContainerStateFn:    func() map[string]structs.ContainerState { return nil },
			UpdateContainerStateFn: func(string, structs.ContainerState) {},
		},
		RuntimeContainerLifecycleStatusOps: orchestration.RuntimeContainerLifecycleStatusOps{
			GetShipStatusFn: func(piers []string) (map[string]string, error) {
				if runtime.getShipStatusFn == nil {
					return nil, nil
				}
				return runtime.getShipStatusFn(piers)
			},
			WaitForShipExitFn: func(patp string, timeout time.Duration) error {
				if runtime.waitForShipExitFn == nil {
					return nil
				}
				return runtime.waitForShipExitFn(patp, timeout)
			},
		},
	}
}

type testRecoveryRuntimeUrbit struct {
	urbitConfFn func(string) structs.UrbitDocker
}

func (runtime testRecoveryRuntimeUrbit) runtimeOps() orchestration.RuntimeUrbitOps {
	return orchestration.RuntimeUrbitOps{
		RuntimeUrbitConfigOps: orchestration.RuntimeUrbitConfigOps{
			LoadUrbitConfigFn: func(string) error { return nil },
			UrbitConfFn: func(patp string) structs.UrbitDocker {
				if runtime.urbitConfFn == nil {
					return structs.UrbitDocker{}
				}
				return runtime.urbitConfFn(patp)
			},
			UpdateUrbitFn: func(string, func(*structs.UrbitDocker) error) error { return nil },
		},
		RuntimeUrbitWorkflowOps: orchestration.RuntimeUrbitWorkflowOps{
			GetContainerNetworkFn: func(string) (string, error) { return "", nil },
			GetLusCodeFn:          func(string) (string, error) { return "", nil },
			ClearLusCodeFn:        func(string) {},
		},
	}
}

type testRecoveryRuntimeConfig struct {
	barExitFn func(string) error
}

func (runtime testRecoveryRuntimeConfig) runtimeOps() orchestration.RuntimeConfigOps {
	return orchestration.RuntimeConfigOps{
		UpdateConfTypedFn: func(...config.ConfUpdateOption) error { return nil },
		WithWgOnFn:        func(enabled bool) config.ConfUpdateOption { return config.WithWgOn(enabled) },
		CycleWgKeyFn:      func() error { return nil },
		BarExitFn: func(patp string) error {
			if runtime.barExitFn == nil {
				return nil
			}
			return runtime.barExitFn(patp)
		},
	}
}

type testRecoveryRuntimeLoad struct {
	loadUrbitsFn func() error
	loadMCFn     func() error
	loadMinIOsFn func() error
}

func (runtime testRecoveryRuntimeLoad) runtimeOps() orchestration.RuntimeLoadOps {
	return orchestration.RuntimeLoadOps{
		LoadWireguardFn: func() error { return nil },
		LoadMCFn: func() error {
			if runtime.loadMCFn == nil {
				return nil
			}
			return runtime.loadMCFn()
		},
		LoadMinIOsFn: func() error {
			if runtime.loadMinIOsFn == nil {
				return nil
			}
			return runtime.loadMinIOsFn()
		},
		LoadUrbitsFn: func() error {
			if runtime.loadUrbitsFn == nil {
				return nil
			}
			return runtime.loadUrbitsFn()
		},
	}
}

func TestRecoverWireguardFleet(t *testing.T) {
	t.Parallel()

	var calls []string
	runtime := orchestration.NewRuntime(
		orchestration.WithContainerOps(testRecoveryRuntimeContainer{
			getShipStatusFn: func(piers []string) (map[string]string, error) {
				calls = append(calls, "get-status:"+strings.Join(piers, ","))
				return map[string]string{
					"sampel": "Up 20 minutes",
					"zod":    "Exited",
				}, nil
			},
			waitForShipExitFn: func(patp string, timeout time.Duration) error {
				calls = append(calls, fmt.Sprintf("wait-exit:%s:%d", patp, timeout))
				return nil
			},
			deleteContainerFn: func(name string) error {
				calls = append(calls, "delete:"+name)
				return nil
			},
			restartContainerFn: func(name string) error {
				calls = append(calls, "restart:"+name)
				return nil
			},
		}.runtimeOps()),
		orchestration.WithUrbitOps(testRecoveryRuntimeUrbit{
			urbitConfFn: func(patp string) structs.UrbitDocker {
				calls = append(calls, "urbit-conf:"+patp)
				if patp == "sampel" {
					return structs.UrbitDocker{
						UrbitNetworkConfig: structs.UrbitNetworkConfig{Network: "wireguard"},
					}
				}
				return structs.UrbitDocker{
					UrbitNetworkConfig: structs.UrbitNetworkConfig{Network: "default"},
				}
			},
		}.runtimeOps()),
		orchestration.WithConfigOps(testRecoveryRuntimeConfig{
			barExitFn: func(patp string) error {
				calls = append(calls, "bar-exit:"+patp)
				return nil
			},
		}.runtimeOps()),
		orchestration.WithLoadOps(testRecoveryRuntimeLoad{
			loadUrbitsFn: func() error {
				calls = append(calls, "load:urbits")
				return nil
			},
			loadMCFn: func() error {
				calls = append(calls, "load:mc")
				return nil
			},
			loadMinIOsFn: func() error {
				calls = append(calls, "load:minios")
				return nil
			},
		}.runtimeOps()),
	)

	if err := RecoverWireguardFleet(NewWireguardRecoveryRuntime(runtime), []string{"sampel", "zod"}, true); err != nil {
		t.Fatalf("RecoverWireguardFleet() unexpected error: %v", err)
	}

	const expectedLen = 12
	if len(calls) != expectedLen {
		t.Fatalf("expected %d calls, got %d", expectedLen, len(calls))
	}
	if !slices.Contains(calls, "wait-exit:sampel:0") {
		t.Fatalf("expected wait-exit call for wireguard ship, got: %v", calls)
	}
	if slices.Contains(calls, "bar-exit:zod") {
		t.Fatalf("did not expect default-network ship to be stopped")
	}
}

func TestRecoverWireguardFleetAggregatesErrors(t *testing.T) {
	t.Parallel()

	var calls []string
	statusErr := errors.New("status down")
	deleteErr := errors.New("delete failed")
	runtime := orchestration.NewRuntime(
		orchestration.WithContainerOps(testRecoveryRuntimeContainer{
			getShipStatusFn: func(piers []string) (map[string]string, error) {
				calls = append(calls, "get-status")
				return map[string]string{"sampel": "Up 2m"}, statusErr
			},
			waitForShipExitFn: func(string, time.Duration) error { return nil },
			deleteContainerFn: func(name string) error {
				calls = append(calls, "delete:"+name)
				return deleteErr
			},
			restartContainerFn: func(string) error { return nil },
		}.runtimeOps()),
		orchestration.WithUrbitOps(testRecoveryRuntimeUrbit{
			urbitConfFn: func(patp string) structs.UrbitDocker {
				calls = append(calls, "urbit-conf")
				return structs.UrbitDocker{
					UrbitNetworkConfig: structs.UrbitNetworkConfig{Network: "wireguard"},
				}
			},
		}.runtimeOps()),
		orchestration.WithLoadOps(testRecoveryRuntimeLoad{
			loadMinIOsFn: func() error {
				calls = append(calls, "load-minios")
				return nil
			},
		}.runtimeOps()),
	)

	err := RecoverWireguardFleet(NewWireguardRecoveryRuntime(runtime), []string{"sampel"}, false)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !strings.Contains(err.Error(), "retrieve ship information") {
		t.Fatalf("expected status error context, got %v", err)
	}
	if !strings.Contains(err.Error(), "delete sampel container") {
		t.Fatalf("expected delete error context, got %v", err)
	}
	if !slices.Contains(calls, "load-minios") {
		t.Fatal("expected minio containers to be loaded after recovery sequence")
	}
	if slices.Contains(calls, "delete:mc") {
		t.Fatal("did not expect delete mc call when deleteMinioClient is false")
	}
}
