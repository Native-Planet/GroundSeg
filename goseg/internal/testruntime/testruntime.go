package testruntime

// Apply returns a copy of base after applying each non-nil mutator.
func Apply[T any](base T, mutators ...func(*T)) T {
	for _, mutate := range mutators {
		if mutate == nil {
			continue
		}
		mutate(&base)
	}
	return base
}
