package shipworkflow

import (
	"fmt"

	"groundseg/broadcast"
	"groundseg/structs"
	"groundseg/transition"
)

type urbitTransitionRuntime struct {
	publishSchedulePack func(string) error
}

var urbitTransitionRuntimeFactory = func() urbitTransitionRuntime {
	return urbitTransitionRuntime{
		publishSchedulePack: broadcast.DefaultBroadcastStateRuntime().PublishSchedulePack,
	}
}

var runTransitionedOperationFn = RunTransitionedOperation

var urbitTransitionRunners = UrbitTransitionRunners()

var urbitTransitionCommandMap = map[string]transition.UrbitTransitionType{
	"set-urbit-domain":       transition.UrbitTransitionUrbitDomain,
	"set-minio-domain":       transition.UrbitTransitionMinIODomain,
	"toggle-chop-on-upgrade": transition.UrbitTransitionChopOnUpgrade,
	"toggle-power":           transition.UrbitTransitionTogglePower,
	"toggle-dev-mode":        transition.UrbitTransitionToggleDevMode,
	"rebuild-container":      transition.UrbitTransitionRebuildContainer,
	"toggle-network":         transition.UrbitTransitionToggleNetwork,
	"toggle-minio-link":      transition.UrbitTransitionToggleMinIOLink,
}

func runUrbitTransitionFromCommand(patp string, transitionType transition.UrbitTransitionType, payload structs.WsUrbitPayload) error {
	runFn, ok := urbitTransitionRunners[transitionType]
	if !ok {
		return runUrbitTransitionFromCommandRegistry(patp, transitionType, payload)
	}
	return runFn(patp, payload)
}

func runUrbitTransitionCommand(patp, command string, payload structs.WsUrbitPayload) error {
	transitionType, ok := urbitTransitionCommandMap[command]
	if !ok {
		return fmt.Errorf("unsupported urbit transition command %q", command)
	}
	return runUrbitTransitionFromCommand(patp, transitionType, payload)
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommand(patp, transition.UrbitTransitionLoom, urbitPayload)
	if err != nil {
		return fmt.Errorf("failed to handle loom transition for %s: %w", patp, err)
	}
	return nil
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload) error {
	err := runUrbitTransitionFromCommand(patp, transition.UrbitTransitionSnapTime, urbitPayload)
	if err != nil {
		return fmt.Errorf("failed to handle snap time transition for %s: %w", patp, err)
	}
	return nil
}
