package testseams

import "testing"

// Override returns a restore function that swaps a target value back to its prior state.
func Override[T any](target *T, value T) func() {
	previous := *target
	*target = value
	return func() {
		*target = previous
	}
}

// WithRestore applies the swap immediately and registers restore in test cleanup.
func WithRestore[T any](t *testing.T, target *T, value T) {
	t.Helper()
	restore := Override(target, value)
	t.Cleanup(func() {
		restore()
	})
}
