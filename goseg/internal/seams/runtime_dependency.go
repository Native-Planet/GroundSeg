package seams

import (
	"errors"
	"fmt"
)

var ErrRuntimeUnconfigured = errors.New("runtime is not configured")

type RuntimeDependencyMissingError struct {
	Dependency string
	Context    string
}

func (err RuntimeDependencyMissingError) Error() string {
	if err.Context == "" {
		return fmt.Sprintf("%s: %s", err.Dependency, ErrRuntimeUnconfigured.Error())
	}
	return fmt.Sprintf("%s: %s (%s)", err.Dependency, ErrRuntimeUnconfigured.Error(), err.Context)
}

func (RuntimeDependencyMissingError) Unwrap() error {
	return ErrRuntimeUnconfigured
}

func MissingRuntimeDependency(dependency, context string) error {
	return RuntimeDependencyMissingError{
		Dependency: dependency,
		Context:    context,
	}
}
