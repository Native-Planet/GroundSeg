package system

import "groundseg/accesspoint"

type accessPointLifecycle interface {
	Start(device string) error
	Stop(device string) error
}

type systemAccessPointLifecycle struct{}

func (systemAccessPointLifecycle) Start(device string) error {
	return accesspoint.Start(device)
}

func (systemAccessPointLifecycle) Stop(device string) error {
	return accesspoint.Stop(device)
}
