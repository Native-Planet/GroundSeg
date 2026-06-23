package docker

import "fmt"

type PowerTransition struct {
	BootStatus       string
	DesiredStatus    string
	UpdateBootStatus bool
	Start            bool
	Stop             bool
}

func PlanPowerTransition(bootStatus string, isRunning bool) (PowerTransition, error) {
	if isRunning {
		switch {
		case IsMaintenanceBootStatus(bootStatus):
			return PowerTransition{}, fmt.Errorf("cannot toggle power while %s maintenance is running", bootStatus)
		case bootStatus == "ignore":
			return PowerTransition{DesiredStatus: "stopped", Stop: true}, nil
		case bootStatus == "boot" || bootStatus == "noboot":
			return PowerTransition{
				BootStatus:       "noboot",
				DesiredStatus:    "stopped",
				UpdateBootStatus: true,
				Stop:             true,
			}, nil
		default:
			return PowerTransition{}, fmt.Errorf("unknown boot status %q", bootStatus)
		}
	}

	switch {
	case IsMaintenanceBootStatus(bootStatus):
		return PowerTransition{
			BootStatus:       "boot",
			DesiredStatus:    "running",
			UpdateBootStatus: true,
			Start:            true,
		}, nil
	case bootStatus == "noboot":
		return PowerTransition{
			BootStatus:       "boot",
			DesiredStatus:    "running",
			UpdateBootStatus: true,
			Start:            true,
		}, nil
	case bootStatus == "boot" || bootStatus == "ignore":
		return PowerTransition{DesiredStatus: "running", Start: true}, nil
	default:
		return PowerTransition{}, fmt.Errorf("unknown boot status %q", bootStatus)
	}
}
