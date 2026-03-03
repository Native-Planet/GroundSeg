package shipworkflow

func HandleStartramServices() error {
	return runStartramServicesWithRuntime(defaultStartramRuntime())
}

func runStartramServicesWithRuntime(runtime startramRuntime) error {
	runtime = resolveStartramRuntime(runtime)
	return runtime.GetStartramServicesFn()
}

func HandleStartramRegions() error {
	return runStartramRegionsWithRuntime(defaultStartramRuntime())
}

func runStartramRegionsWithRuntime(runtime startramRuntime) error {
	runtime = resolveStartramRuntime(runtime)
	return runtime.LoadStartramRegionsFn()
}

func HandleStartramRegister(regCode, region string) error {
	return runStartramRegisterWithRuntime(defaultStartramRuntime(), regCode, region)
}
