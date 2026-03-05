package shipworkflow

import (
	"fmt"
	"strings"
	"time"

	"groundseg/docker/orchestration"
	"groundseg/internal/seams"
	"groundseg/internal/workflow"
	"groundseg/structs"
)

// wireguardRecoveryRuntime is a small seam for wireguard recovery orchestration.
type wireguardRecoveryRuntime struct {
	GetShipStatusFn    func([]string) (map[string]string, error)
	UrbitConfFn        func(string) structs.UrbitDocker
	RestartContainerFn func(string) error
	BarExitFn          func(string) error
	WaitForShipExitFn  func(string, time.Duration) error
	DeleteContainerFn  func(string) error
	LoadUrbitsFn       func() error
	LoadMCFn           func() error
	LoadMinIOsFn       func() error
}

// NewWireguardRecoveryRuntime constructs the recovery runtime seam from orchestration runtime functions.
func NewWireguardRecoveryRuntime(runtime orchestration.Runtime) wireguardRecoveryRuntime {
	return wireguardRecoveryRuntime{
		GetShipStatusFn:    runtime.GetShipStatusFn,
		UrbitConfFn:        runtime.UrbitConfFn,
		RestartContainerFn: runtime.RestartContainerFn,
		BarExitFn:          runtime.BarExitFn,
		WaitForShipExitFn:  runtime.WaitForShipExitFn,
		DeleteContainerFn:  runtime.DeleteContainerFn,
		LoadUrbitsFn:       runtime.LoadUrbitsFn,
		LoadMCFn:           runtime.LoadMCFn,
		LoadMinIOsFn:       runtime.LoadMinIOsFn,
	}
}

func (runtime wireguardRecoveryRuntime) validate() error {
	if runtime.GetShipStatusFn == nil {
		return seams.MissingRuntimeDependency("wireguard ship status callback", "")
	}
	if runtime.UrbitConfFn == nil {
		return seams.MissingRuntimeDependency("wireguard urbit config callback", "")
	}
	if runtime.RestartContainerFn == nil {
		return seams.MissingRuntimeDependency("wireguard restart container callback", "")
	}
	if runtime.BarExitFn == nil {
		return seams.MissingRuntimeDependency("wireguard bar exit callback", "")
	}
	if runtime.WaitForShipExitFn == nil {
		return seams.MissingRuntimeDependency("wireguard wait for ship exit callback", "")
	}
	if runtime.DeleteContainerFn == nil {
		return seams.MissingRuntimeDependency("wireguard delete container callback", "")
	}
	if runtime.LoadUrbitsFn == nil {
		return seams.MissingRuntimeDependency("wireguard load urbits callback", "")
	}
	if runtime.LoadMCFn == nil {
		return seams.MissingRuntimeDependency("wireguard load mainchain callback", "")
	}
	if runtime.LoadMinIOsFn == nil {
		return seams.MissingRuntimeDependency("wireguard load minio callback", "")
	}
	return nil
}

// RecoverWireguardFleet performs wireguard restart/recovery orchestration with consistent error accumulation.
func RecoverWireguardFleet(runtime wireguardRecoveryRuntime, piers []string, deleteMinioClient bool) error {
	if err := runtime.validate(); err != nil {
		return err
	}
	wgShips := map[string]bool{}
	steps := []workflow.Step{}

	shipStatus, err := runtime.GetShipStatusFn(piers)
	if err != nil {
		steps = append(steps, workflow.Step{
			Name: "retrieve ship information",
			Run:  func() error { return err },
		})
	}
	for pier, status := range shipStatus {
		dockerConfig := runtime.UrbitConfFn(pier)
		if dockerConfig.Network == "wireguard" {
			wgShips[pier] = status == "Up" || strings.HasPrefix(status, "Up ")
		}
	}

	steps = append(steps, workflow.Step{
		Name: "restart wireguard",
		Run: func() error {
			return runtime.RestartContainerFn("wireguard")
		},
	})

	wgPiers := make([]string, 0, len(wgShips))
	for patp, isRunning := range wgShips {
		pirate := patp
		if isRunning {
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("stop %s with |exit before restart", pirate),
				Run:  func() error { return runtime.BarExitFn(pirate) },
			})
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("wait for %s exit before restart", pirate),
				Run: func() error {
					return runtime.WaitForShipExitFn(pirate, 0)
				},
			})
		}
		wgPiers = append(wgPiers, patp)
	}
	steps = appendShipContainerRebuildSteps(steps, shipContainerRebuildRuntime{
		DeleteContainerFn: runtime.DeleteContainerFn,
		LoadUrbitsFn:      runtime.LoadUrbitsFn,
		LoadMCFn:          runtime.LoadMCFn,
		LoadMinIOsFn:      runtime.LoadMinIOsFn,
	}, shipContainerRebuildOptions{
		piers:             wgPiers,
		deletePiers:       true,
		deleteMinioClient: deleteMinioClient,
		loadUrbits:        true,
		loadMinIOClient:   true,
		loadMinIOs:        true,
	})

	if joined := runOrchestrationSteps(steps, "wireguard recovery"); joined != nil {
		return joined
	}
	return nil
}
