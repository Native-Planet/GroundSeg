package docker

import "groundseg/structs"

var (
	urbitTransitionBus      = make(chan structs.UrbitTransition, 100)
	systemTransitionBus     = make(chan structs.SystemTransition, 100)
	newShipTransitionBus    = make(chan structs.NewShipTransition, 100)
	importShipTransitionBus = make(chan structs.UploadTransition, 100)
)

func PublishUrbitTransition(event structs.UrbitTransition) {
	urbitTransitionBus <- event
}

func UrbitTransitions() <-chan structs.UrbitTransition {
	return urbitTransitionBus
}

func PublishSystemTransition(event structs.SystemTransition) {
	systemTransitionBus <- event
}

func SystemTransitions() <-chan structs.SystemTransition {
	return systemTransitionBus
}

func PublishNewShipTransition(event structs.NewShipTransition) {
	newShipTransitionBus <- event
}

func NewShipTransitions() <-chan structs.NewShipTransition {
	return newShipTransitionBus
}

func PublishImportShipTransition(event structs.UploadTransition) {
	importShipTransitionBus <- event
}

func ImportShipTransitions() <-chan structs.UploadTransition {
	return importShipTransitionBus
}
