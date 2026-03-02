package shipworkflow

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/startram"
	"groundseg/structs"
	"strings"
	"time"
)

func WaitForBootCode(patp string, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		code, err := click.GetLusCode(patp)
		if err != nil {
			continue
		}
		if len(code) == 27 {
			return
		}
	}
}

func WaitForRemoteReady(patp string, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		tramConf := config.GetStartramConfig()
		found := false
		ready := true
		for _, subdomain := range tramConf.Subdomains {
			if strings.Contains(subdomain.URL, patp) {
				found = true
				if subdomain.Status != "ok" {
					ready = false
					break
				}
			}
		}
		if !found || ready {
			return
		}
	}
}

func RegisterShipServices(patp string) error {
	if err := startram.RegisterNewShip(patp); err != nil {
		return fmt.Errorf("unable to register startram service for %s: %w", patp, err)
	}
	return nil
}

func SwitchShipToWireguard(patp string, gracefulStop bool) error {
	if err := config.UpdateUrbit(patp, func(shipConf *structs.UrbitDocker) error {
		shipConf.Network = "wireguard"
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update urbit config for %s: %w", patp, err)
	}

	if gracefulStop {
		statuses, err := orchestration.GetShipStatus([]string{patp})
		if err != nil {
			return fmt.Errorf("failed to get statuses for %s when rebuilding container: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return fmt.Errorf("%s status doesn't exist", patp)
		}
		if strings.Contains(status, "Up") {
			if err := click.BarExit(patp); err != nil {
				return fmt.Errorf("failed to stop %s with |exit for rebuilding container: %w", patp, err)
			}
		}
	}

	if err := orchestration.DeleteContainer(patp); err != nil {
		// keep going; this can fail when container has already exited/been removed.
	}

	if _, err := orchestration.StartContainer("minio_"+patp, "minio"); err != nil {
		// keep going; minio startup is best effort here and can be retried by health loops.
	}

	info, err := orchestration.StartContainer(patp, "vere")
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", patp, err)
	}
	config.UpdateContainerState(patp, info)
	return nil
}
