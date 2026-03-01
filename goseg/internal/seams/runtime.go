package seams

import "groundseg/config"

type RuntimeBase struct {
	BasePath     func() string
	Architecture func() string
	DebugMode    func() bool
}

func NewRuntimeBase() RuntimeBase {
	return RuntimeBase{
		BasePath:     func() string { return config.BasePath },
		Architecture: func() string { return config.Architecture },
		DebugMode:    func() bool { return config.DebugMode },
	}
}

type DockerRuntimeBase struct {
	RuntimeBase
	DockerDir func() string
}

func NewDockerRuntimeBase() DockerRuntimeBase {
	return DockerRuntimeBase{
		RuntimeBase: NewRuntimeBase(),
		DockerDir:   func() string { return config.DockerDir },
	}
}
