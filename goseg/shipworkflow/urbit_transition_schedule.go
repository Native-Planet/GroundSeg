package shipworkflow

import (
	"fmt"

	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/structs"
	"groundseg/transition"
)

func schedulePack(patp string, urbitPayload structs.WsUrbitPayload) error {
	runtime := urbitTransitionRuntimeFactory()
	frequency := urbitPayload.Payload.Frequency
	if frequency < 1 {
		return fmt.Errorf("pack frequency must be greater than zero")
	}
	intervalType := urbitPayload.Payload.IntervalType
	switch intervalType {
	case "month", "week", "day":
		if err := persistShipUrbitSectionConfig[structs.UrbitScheduleConfig](patp, dockerOrchestration.UrbitConfigSectionSchedule, func(conf *structs.UrbitScheduleConfig) error {
			conf.MeldTime = urbitPayload.Payload.Time
			conf.MeldSchedule = true
			conf.MeldScheduleType = intervalType
			conf.MeldFrequency = frequency
			conf.MeldDay = urbitPayload.Payload.Day
			conf.MeldDate = urbitPayload.Payload.Date
			return nil
		}); err != nil {
			return fmt.Errorf("failed to update pack schedule: %w", err)
		}
	default:
		return fmt.Errorf("unknown pack schedule interval type: %s", intervalType)
	}
	if err := runtime.publishSchedulePack("schedule"); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("publish schedule-pack transition for %s", patp),
			err,
			transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		)
	}
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload) error {
	_ = urbitPayload
	if err := persistShipUrbitSectionConfig[structs.UrbitScheduleConfig](patp, dockerOrchestration.UrbitConfigSectionSchedule, func(conf *structs.UrbitScheduleConfig) error {
		conf.MeldSchedule = false
		return nil
	}); err != nil {
		return fmt.Errorf("failed to pause pack schedule: %w", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipUrbitSectionConfig[structs.UrbitRuntimeConfig](patp, dockerOrchestration.UrbitConfigSectionRuntime, func(conf *structs.UrbitRuntimeConfig) error {
		conf.SizeLimit = urbitPayload.Payload.Value
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set new size limit for %s: %w", patp, err)
	}
	return nil
}
