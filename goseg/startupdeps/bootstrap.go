package startupdeps

import (
	"groundseg/broadcast"
	"groundseg/docker/orchestration"
)

type StartupDockerRuntime interface {
	Initialize() error
}

func NewStartupDockerRuntime() StartupDockerRuntime {
	return orchestration.NewStartupRuntime()
}

func InitializeBroadcast() error {
	return broadcast.Initialize()
}

func InitializeDockerRuntime() func() error {
	return NewStartupDockerRuntime().Initialize
}
