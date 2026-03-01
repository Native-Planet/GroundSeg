package shipworkflow

import "groundseg/structs"

func InstallPenpaiCompanion(patp string) error {
	return installPenpaiCompanion(patp)
}

func UninstallPenpaiCompanion(patp string) error {
	return uninstallPenpaiCompanion(patp)
}

func InstallGallseg(patp string) error {
	return installGallseg(patp)
}

func UninstallGallseg(patp string) error {
	return uninstallGallseg(patp)
}

func StartramReminder(patp string, remind bool) error {
	return startramReminder(patp, remind)
}

func UrbitDeleteStartramService(patp string, service string) error {
	return urbitDeleteStartramService(patp, service)
}

func PackPier(patp string) error {
	return packPier(patp)
}

func PackMeldPier(patp string) error {
	return packMeldPier(patp)
}

func ToggleAlias(patp string) error {
	return toggleAlias(patp)
}

func SetUrbitDomain(patp string, payload structs.WsUrbitPayload) error {
	return setUrbitDomain(patp, payload)
}

func SetMinIODomain(patp string, payload structs.WsUrbitPayload) error {
	return setMinIODomain(patp, payload)
}

func ToggleChopOnVereUpdate(patp string) error {
	return toggleChopOnVereUpdate(patp)
}

func DeleteShip(patp string) error {
	return deleteShip(patp)
}

func ExportShip(patp string, payload structs.WsUrbitPayload) error {
	return exportShip(patp, payload)
}

func ExportBucket(patp string, payload structs.WsUrbitPayload) error {
	return exportBucket(patp, payload)
}

func TogglePower(patp string) error {
	return togglePower(patp)
}

func ToggleDevMode(patp string) error {
	return toggleDevMode(patp)
}

func RebuildContainer(patp string) error {
	return rebuildContainer(patp)
}

func ToggleNetwork(patp string) error {
	return toggleNetwork(patp)
}

func ToggleBootStatus(patp string) error {
	return toggleBootStatus(patp)
}

func ToggleAutoReboot(patp string) error {
	return toggleAutoReboot(patp)
}

func ToggleMinIOLink(patp string) error {
	return toggleMinIOLink(patp)
}

func SetLoom(patp string, payload structs.WsUrbitPayload) error {
	return handleLoom(patp, payload)
}

func SetSnapTime(patp string, payload structs.WsUrbitPayload) error {
	return handleSnapTime(patp, payload)
}

func SchedulePack(patp string, payload structs.WsUrbitPayload) error {
	return schedulePack(patp, payload)
}

func PausePackSchedule(patp string, payload structs.WsUrbitPayload) error {
	return pausePackSchedule(patp, payload)
}

func SetNewMaxPierSize(patp string, payload structs.WsUrbitPayload) error {
	return setNewMaxPierSize(patp, payload)
}

func RollChopPier(patp string) error {
	return rollChopPier(patp)
}
