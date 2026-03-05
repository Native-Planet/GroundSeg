package contextapi

import "groundseg/config/runtimecontext"

type RuntimeContext = runtimecontext.RuntimeContext

func Set(context RuntimeContext) {
	runtimecontext.Set(context)
}

func Snapshot() RuntimeContext {
	return runtimecontext.Snapshot()
}

func BasePath() string {
	return runtimecontext.BasePath()
}

func Architecture() string {
	return runtimecontext.Architecture()
}

func DebugMode() bool {
	return runtimecontext.DebugMode()
}

func DockerDir() string {
	return runtimecontext.DockerDir()
}

func SetBasePath(basePath string) {
	runtimecontext.SetBasePath(basePath)
}

func SetArchitecture(architecture string) {
	runtimecontext.SetArchitecture(architecture)
}

func SetDebugMode(debugMode bool) {
	runtimecontext.SetDebugMode(debugMode)
}

func SetDockerDir(dockerDir string) {
	runtimecontext.SetDockerDir(dockerDir)
}
