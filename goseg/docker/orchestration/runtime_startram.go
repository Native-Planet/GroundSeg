package orchestration

import "groundseg/internal/seams"

// StartramRuntime aliases the shared orchestration runtime with startram-specific
// dependency fields used by service registration workflows.
type StartramRuntime = Runtime

type StartramRuntimeOption = RuntimeOption

func WithStartramServiceLoaders(getStartramServicesFn, loadStartramRegionsFn func() error) RuntimeOption {
	return WithStartramOps(RuntimeStartramOps{
		GetStartramServicesFn: func() error {
			if getStartramServicesFn == nil {
				return nil
			}
			return getStartramServicesFn()
		},
		LoadStartramRegionsFn: func() error {
			if loadStartramRegionsFn == nil {
				return nil
			}
			return loadStartramRegionsFn()
		},
	})
}

// NewStartramRuntime builds a runtime that includes the startram dependency graph
// required by ship workflow handlers.
func NewStartramRuntime(opts ...StartramRuntimeOption) StartramRuntime {
	return NewRuntime(opts...)
}

// NewStartramRuntimeWithDefaults builds a runtime and applies a first-pass default
// set for startram dependency seams. Provided opts always override defaults.
func NewStartramRuntimeWithDefaults(defaultOps RuntimeStartramOps, opts ...StartramRuntimeOption) StartramRuntime {
	runtime := NewRuntime(opts...)
	runtime.RuntimeStartramOps = seams.WithDefaults(runtime.RuntimeStartramOps, defaultOps)
	return runtime
}

// ResolveStartramRuntime applies default startram seams to an existing runtime seam
// collection without mutating the provided runtime.
func ResolveStartramRuntime(runtime StartramRuntime, defaultOps RuntimeStartramOps) StartramRuntime {
	return NewStartramRuntimeWithDefaults(defaultOps, WithRuntimeDependencies(runtime))
}
