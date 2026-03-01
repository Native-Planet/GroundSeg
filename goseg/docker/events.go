package docker

import (
	"groundseg/docker/events"
	"groundseg/structs"
)

func PublishUrbitTransition(event structs.UrbitTransition) {
	events.PublishUrbitTransition(event)
}

func UrbitTransitions() <-chan structs.UrbitTransition {
	return events.UrbitTransitions()
}

func PublishSystemTransition(event structs.SystemTransition) {
	events.PublishSystemTransition(event)
}

func SystemTransitions() <-chan structs.SystemTransition {
	return events.SystemTransitions()
}

func PublishNewShipTransition(event structs.NewShipTransition) {
	events.PublishNewShipTransition(event)
}

func NewShipTransitions() <-chan structs.NewShipTransition {
	return events.NewShipTransitions()
}

func PublishImportShipTransition(event structs.UploadTransition) {
	events.PublishImportShipTransition(event)
}

func ImportShipTransitions() <-chan structs.UploadTransition {
	return events.ImportShipTransitions()
}
