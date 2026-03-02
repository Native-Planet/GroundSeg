package workflow

import (
	"errors"
	"fmt"
)

// Step represents a named execution unit with optional failure context.
type Step struct {
	// Name is attached to any failure returned by Run.
	Name string
	// Run performs the step.
	Run func() error
}

func wrapStepError(name string, err error) error {
	if name == "" {
		return err
	}
	return fmt.Errorf("%s: %w", name, err)
}

// Collect executes every step and returns wrapped errors while continuing on failure.
func Collect(steps []Step, onFailure func(error)) []error {
	if onFailure == nil {
		onFailure = func(error) {}
	}

	var errs []error
	for _, step := range steps {
		if step.Run == nil {
			continue
		}
		if err := step.Run(); err != nil {
			err = wrapStepError(step.Name, err)
			onFailure(err)
			errs = append(errs, err)
		}
	}
	return errs
}

// Join executes every step and returns errors.Join over wrapped failures.
func Join(steps []Step, onFailure func(error)) error {
	return errors.Join(Collect(steps, onFailure)...)
}
