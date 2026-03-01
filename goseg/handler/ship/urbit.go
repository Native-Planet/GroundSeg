package ship

import (
	"encoding/json"
	"fmt"
	"groundseg/chopsvc"
	"groundseg/click"
	"groundseg/docker"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

var (
	urbitGetShipStatus     = docker.GetShipStatus
	urbitDeleteContainer   = docker.DeleteContainer
	urbitBarExit           = click.BarExit
	urbitSleep             = time.Sleep
	waitCompletePoller     = shipworkflow.PollWithTimeout
	areSubdomainsAliasesFn = shipworkflow.AreSubdomainsAliases
	waitCompleteFn         = func(patp string) error {
		return shipworkflow.WaitForUrbitStop(patp, urbitGetShipStatus, waitCompletePoller)
	}
	urbitCleanDeleteFn = shipworkflow.UrbitCleanDelete
)

type urbitCommand func(patp string, payload structs.WsUrbitPayload) error

func urbitWithShip(handler func(string) error) urbitCommand {
	return func(patp string, _ structs.WsUrbitPayload) error {
		return handler(patp)
	}
}

func urbitWithPayloadAndShip(handler func(string, structs.WsUrbitPayload) error) urbitCommand {
	return func(patp string, payload structs.WsUrbitPayload) error {
		return handler(patp, payload)
	}
}

func urbitWithPatp(handler func(string) error) urbitCommand {
	return func(patp string, _ structs.WsUrbitPayload) error {
		return handler(patp)
	}
}

var urbitCommands = map[string]urbitCommand{
	"toggle-alias":     urbitWithShip(shipworkflow.ToggleAlias),
	"export-bucket":    urbitWithPayloadAndShip(shipworkflow.ExportBucket),
	"export-ship":      urbitWithPayloadAndShip(shipworkflow.ExportShip),
	"set-urbit-domain": urbitWithPayloadAndShip(shipworkflow.SetUrbitDomain),
	"set-minio-domain": urbitWithPayloadAndShip(shipworkflow.SetMinIODomain),
	"startram-reminder": func(patp string, payload structs.WsUrbitPayload) error {
		return shipworkflow.StartramReminder(patp, payload.Payload.Remind)
	},
	"delete-service": func(patp string, payload structs.WsUrbitPayload) error {
		return shipworkflow.UrbitDeleteStartramService(patp, payload.Payload.Service)
	},
	"install-penpai-companion":   urbitWithShip(shipworkflow.InstallPenpaiCompanion),
	"uninstall-penpai-companion": urbitWithShip(shipworkflow.UninstallPenpaiCompanion),
	"install-gallseg":            urbitWithShip(shipworkflow.InstallGallseg),
	"uninstall-gallseg":          urbitWithShip(shipworkflow.UninstallGallseg),
	"chop":                       urbitWithShip(chopsvc.ChopPier),
	"roll-chop":                  urbitWithShip(shipworkflow.RollChopPier),
	"pack":                       urbitWithShip(shipworkflow.PackPier),
	"pack-meld":                  urbitWithShip(shipworkflow.PackMeldPier),
	"rebuild-container":          urbitWithShip(shipworkflow.RebuildContainer),
	"toggle-minio-link":          urbitWithShip(shipworkflow.ToggleMinIOLink),
	"pause-pack-schedule":        urbitWithPayloadAndShip(shipworkflow.PausePackSchedule),
	"schedule-pack":              urbitWithPayloadAndShip(shipworkflow.SchedulePack),
	"loom":                       urbitWithPayloadAndShip(shipworkflow.SetLoom),
	"snaptime":                   urbitWithPayloadAndShip(shipworkflow.SetSnapTime),
	"toggle-boot-status":         urbitWithShip(shipworkflow.ToggleBootStatus),
	"toggle-auto-reboot":         urbitWithShip(shipworkflow.ToggleAutoReboot),
	"toggle-network":             urbitWithShip(shipworkflow.ToggleNetwork),
	"toggle-devmode":             urbitWithShip(shipworkflow.ToggleDevMode),
	"toggle-power":               urbitWithShip(shipworkflow.TogglePower),
	"delete-ship":                urbitWithShip(shipworkflow.DeleteShip),
	"toggle-chop-on-vere-update": urbitWithShip(shipworkflow.ToggleChopOnVereUpdate),
	"new-max-pier-size":          urbitWithPayloadAndShip(shipworkflow.SetNewMaxPierSize),
	"toggle-backup":              urbitWithShip(handleLocalToggleBackup),
	"toggle-startram-backup":     urbitWithShip(handleStartramToggleBackup),
	"local-backup":               urbitWithPatp(handleLocalBackup),
	"schedule-local-backup":       urbitWithPayloadAndShip(handleScheduleLocalBackup),
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
	handler, exists := urbitCommands[urbitPayload.Payload.Action]
	if !exists {
		return fmt.Errorf("unrecognized urbit action: %v", urbitPayload.Payload.Action)
	}
	return handler(patp, urbitPayload)
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
	return areSubdomainsAliasesFn(domain1, domain2)
}

func WaitComplete(patp string) error {
	return waitCompleteFn(patp)
}

func urbitCleanDelete(patp string) error {
	return urbitCleanDeleteFn(patp)
}
