package seams

// ResolveRuntime composes defaults with an optional override and validates the
// resulting runtime in one place.
func ResolveRuntime[T any](defaults T, validate func(T) error, overrides ...T) (T, error) {
	resolved := defaults
	if len(overrides) > 0 {
		resolved = Merge(defaults, overrides[0])
	}
	if validate != nil {
		if err := validate(resolved); err != nil {
			return resolved, err
		}
	}
	return resolved, nil
}
