package system

import (
	"groundseg/docker/orchestration"
	"groundseg/shipworkflow"
)

// RecoverWireguardFleet performs the shared wireguard restart/recovery orchestration
// used by both manual StarTram restart and automatic 502-recovery paths.
func RecoverWireguardFleet(piers []string, deleteMinioClient bool) error {
	return shipworkflow.RecoverWireguardFleet(shipworkflow.NewWireguardRecoveryRuntime(orchestration.NewRuntime()), piers, deleteMinioClient)
}
