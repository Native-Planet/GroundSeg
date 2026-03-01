package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	lookupCNAME          = net.LookupCNAME
	urbitGetShipStatus   = docker.GetShipStatus
	urbitDeleteContainer = docker.DeleteContainer
	urbitBarExit         = click.BarExit
	urbitSleep           = time.Sleep
	waitCompletePoller   = pollWithTimeout
)

type urbitCommand func(patp string, payload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error

func urbitWithShip(handler func(string, structs.UrbitDocker) error) urbitCommand {
	return func(patp string, _ structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
		return handler(patp, shipConf)
	}
}

func urbitWithPayloadAndShip(handler func(string, structs.WsUrbitPayload, structs.UrbitDocker) error) urbitCommand {
	return func(patp string, payload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
		return handler(patp, payload, shipConf)
	}
}

func urbitWithPatp(handler func(string) error) urbitCommand {
	return func(patp string, _ structs.WsUrbitPayload, _ structs.UrbitDocker) error {
		return handler(patp)
	}
}

var urbitCommands = map[string]urbitCommand{
	"toggle-alias":     urbitWithShip(toggleAlias),
	"export-bucket":    urbitWithPayloadAndShip(exportBucket),
	"export-ship":      urbitWithPayloadAndShip(exportShip),
	"set-urbit-domain": urbitWithPayloadAndShip(setUrbitDomain),
	"set-minio-domain": urbitWithPayloadAndShip(setMinIODomain),
	"startram-reminder": func(patp string, payload structs.WsUrbitPayload, _ structs.UrbitDocker) error {
		return startramReminder(patp, payload.Payload.Remind)
	},
	"delete-service": func(patp string, payload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
		return urbitDeleteStartramService(patp, payload.Payload.Service, shipConf)
	},
	"install-penpai-companion":   urbitWithShip(installPenpaiCompanion),
	"uninstall-penpai-companion": urbitWithShip(uninstallPenpaiCompanion),
	"install-gallseg":            urbitWithShip(installGallseg),
	"uninstall-gallseg":          urbitWithShip(uninstallGallseg),
	"chop":                       urbitWithShip(ChopPier),
	"roll-chop":                  urbitWithShip(rollChopPier),
	"pack":                       urbitWithShip(packPier),
	"pack-meld":                  urbitWithShip(packMeldPier),
	"rebuild-container":          urbitWithShip(rebuildContainer),
	"toggle-minio-link":          urbitWithShip(toggleMinIOLink),
	"pause-pack-schedule":        urbitWithPayloadAndShip(pausePackSchedule),
	"schedule-pack":              urbitWithPayloadAndShip(schedulePack),
	"loom":                       urbitWithPayloadAndShip(handleLoom),
	"snaptime":                   urbitWithPayloadAndShip(handleSnapTime),
	"toggle-boot-status":         urbitWithShip(toggleBootStatus),
	"toggle-auto-reboot":         urbitWithShip(toggleAutoReboot),
	"toggle-network":             urbitWithShip(toggleNetwork),
	"toggle-devmode":             urbitWithShip(toggleDevMode),
	"toggle-power":               urbitWithShip(togglePower),
	"delete-ship":                urbitWithShip(deleteShip),
	"toggle-chop-on-vere-update": urbitWithShip(toggleChopOnVereUpdate),
	"new-max-pier-size":          urbitWithPayloadAndShip(setNewMaxPierSize),
	"toggle-backup":              urbitWithShip(handleLocalToggleBackup),
	"toggle-startram-backup":     urbitWithShip(handleStartramToggleBackup),
	"local-backup":               urbitWithPatp(handleLocalBackup),
	"schedule-local-backup":      urbitWithPayloadAndShip(handleScheduleLocalBackup),
	"restore-tlon-backup":        urbitWithPayloadAndShip(handleRestoreTlonBackup),
}

// handle urbit-type events
func UrbitHandler(msg []byte) error {
	zap.L().Info("Urbit")
	var urbitPayload structs.WsUrbitPayload
	err := json.Unmarshal(msg, &urbitPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal urbit payload: %w", err)
	}
	patp := urbitPayload.Payload.Patp
	shipConf := config.UrbitConf(patp)
	handler, exists := urbitCommands[urbitPayload.Payload.Action]
	if !exists {
		return fmt.Errorf("unrecognized urbit action: %v", urbitPayload.Payload.Action)
	}
	return handler(patp, urbitPayload, shipConf)
}

// remove a string from a slice of strings
func cutSlice(slice []string, s string) []string {
	index := -1
	for i, v := range slice {
		if v == s {
			index = i
			break
		}
	}
	if index == -1 {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

// AreSubdomainsAliases checks if two subdomains are aliases of each other.
func AreSubdomainsAliases(domain1, domain2 string) (bool, error) {
	// Skip check for alt domains
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("Invalid subdomain")
	}
	if config.GetStartramConfig().Cname != "" && domain1[firstDot+1:] == config.GetStartramConfig().Cname {
		// if it matches startram alt cname, we good
		return true, nil
	}
	// Lookup CNAME for the first domain
	cname1, err := lookupCNAME(domain1)
	if err != nil {
		return false, err
	}

	// Lookup CNAME for the second domain
	cname2, err := lookupCNAME(domain2)
	if err != nil {
		return false, err
	}

	// Compare CNAMEs
	return cname1 == cname2, nil
}

func WaitComplete(patp string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	const maxStatusFailures = 5
	statusFailureCount := 0
	err := waitCompletePoller(ctx, 500*time.Millisecond, func() (bool, error) {
		statuses, err := urbitGetShipStatus([]string{patp})
		if err != nil {
			statusFailureCount++
			wrappedErr := fmt.Errorf("retrieve ship status for %s: %w", patp, err)
			if statusFailureCount >= maxStatusFailures {
				return false, wrappedErr
			}
			zap.L().Warn(fmt.Sprintf("Retrying wait-complete status retrieval for %s (%d/%d): %v", patp, statusFailureCount, maxStatusFailures, wrappedErr))
			return false, nil
		}
		status, exists := statuses[patp]
		if !exists {
			statusFailureCount++
			missingErr := fmt.Errorf("status for %s not found", patp)
			if statusFailureCount >= maxStatusFailures {
				return false, missingErr
			}
			zap.L().Warn(fmt.Sprintf("Retrying wait-complete status lookup for %s (%d/%d): %v", patp, statusFailureCount, maxStatusFailures, missingErr))
			return false, nil
		}
		statusFailureCount = 0
		if strings.Contains(status, "Up") {
			zap.L().Debug(fmt.Sprintf("%s continue waiting...", patp))
			return false, nil
		}
		zap.L().Debug(fmt.Sprintf("%s finished", patp))
		return true, nil
	})
	if err == context.DeadlineExceeded {
		zap.L().Warn(fmt.Sprintf("%s timed out waiting for completion", patp))
		return err
	}
	if err != nil {
		zap.L().Error(fmt.Sprintf("%s wait-complete failed: %v", patp, err))
		return err
	}
	return nil
}

func persistShipConf(patp string, shipConf structs.UrbitDocker) error {
	return config.UpdateUrbit(patp, func(conf *structs.UrbitDocker) error {
		*conf = shipConf
		return nil
	})
}

func urbitCleanDelete(patp string) error {
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := urbitGetShipStatus([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	status, err := getShipRunningStatus(patp)
	if err == nil {
		isRunning := strings.Contains(status, "Up")
		if isRunning {
			if err := urbitBarExit(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit: %v", patp, err))
			}
		}
		for {
			status, err := getShipRunningStatus(patp)
			if err != nil {
				break
			}
			zap.L().Debug(fmt.Sprintf("%s", status))
			if !strings.Contains(status, "Up") {
				break
			}
			urbitSleep(1 * time.Second)
		}
	}
	if err := urbitDeleteContainer(patp); err != nil {
		return fmt.Errorf("Failed to delete container %s: %w", patp, err)
	}
	return nil
}
