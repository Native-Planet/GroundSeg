package api

import (
	"groundseg/shipworkflow"
)

var (
	handleNewShipBoot   = shipworkflow.HandleNewShipBoot
	handleNewShipCancel = shipworkflow.CancelNewShip
	handleNewShipReset  = shipworkflow.ResetNewShip
)
