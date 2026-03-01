package api

import (
	"groundseg/handler/ship"
	"groundseg/handler/system"
)

// InitializeSupport initializes support report machinery.
func InitializeSupport() {
	system.InitializeSupport()
}

// UrbitHandler dispatches URbit actions.
func UrbitHandler(msg []byte) error {
	return ship.UrbitHandler(msg)
}

// PenpaiHandler dispatches Penpai actions.
func PenpaiHandler(msg []byte) error {
	return system.PenpaiHandler(msg)
}

// SystemHandler dispatches system actions.
func SystemHandler(msg []byte) error {
	return system.SystemHandler(msg)
}

// StartramHandler dispatches StarTram actions.
func StartramHandler(msg []byte) error {
	return system.StartramHandler(msg)
}

// SupportHandler handles bug report submission.
func SupportHandler(msg []byte) error {
	return system.SupportHandler(msg)
}

// WaitComplete waits for a ship container to stop.
func WaitComplete(patp string) error {
	return ship.WaitComplete(patp)
}

// RecoverWireguardFleet performs the shared wireguard restart/recovery flow.
func RecoverWireguardFleet(piers []string, deleteMinioClient bool) error {
	return system.RecoverWireguardFleet(piers, deleteMinioClient)
}
