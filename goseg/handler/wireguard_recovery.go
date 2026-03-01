package handler

import (
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"strings"
)

// RecoverWireguardFleet performs the shared wireguard restart/recovery orchestration
// used by both manual StarTram restart and automatic 502-recovery paths.
func RecoverWireguardFleet(piers []string, deleteMinioClient bool) error {
	var stepErrors []error

	wgShips := map[string]bool{}
	pierStatus, err := docker.GetShipStatus(piers)
	if err != nil {
		appendOrchestrationError(&stepErrors, "retrieve ship information", err)
	}
	for pier, status := range pierStatus {
		dockerConfig := config.UrbitConf(pier)
		if dockerConfig.Network == "wireguard" {
			wgShips[pier] = (status == "Up" || strings.HasPrefix(status, "Up "))
		}
	}

	if err := docker.RestartContainer("wireguard"); err != nil {
		appendOrchestrationError(&stepErrors, "restart Wireguard", err)
	}

	for patp, isRunning := range wgShips {
		if isRunning {
			if err := click.BarExit(patp); err != nil {
				appendOrchestrationError(&stepErrors, fmt.Sprintf("stop %s with |exit before restart", patp), err)
			} else {
				if err := docker.WaitForShipExit(patp, 0); err != nil {
					appendOrchestrationError(&stepErrors, fmt.Sprintf("wait for %s exit before restart", patp), err)
				}
			}
		}
		if err := docker.DeleteContainer(patp); err != nil {
			appendOrchestrationError(&stepErrors, fmt.Sprintf("delete %s container", patp), err)
		}
		minio := fmt.Sprintf("minio_%s", patp)
		if err := docker.DeleteContainer(minio); err != nil {
			appendOrchestrationError(&stepErrors, fmt.Sprintf("delete %s container", minio), err)
		}
	}

	if deleteMinioClient {
		if err := docker.DeleteContainer("mc"); err != nil {
			appendOrchestrationError(&stepErrors, "delete minio client container", err)
		}
	}

	if err := docker.LoadUrbits(); err != nil {
		appendOrchestrationError(&stepErrors, "load urbit containers", err)
	}
	if err := docker.LoadMC(); err != nil {
		appendOrchestrationError(&stepErrors, "load minio client container", err)
	}
	if err := docker.LoadMinIOs(); err != nil {
		appendOrchestrationError(&stepErrors, "load minio containers", err)
	}

	return errors.Join(stepErrors...)
}
