package errpolicy

import "fmt"

// MaskedError preserves an internal cause chain while exposing a safe outward message.
type MaskedError struct {
	message string
	cause   error
}

func (e *MaskedError) Error() string {
	return e.message
}

func (e *MaskedError) Unwrap() error {
	return e.cause
}

func WrapMasked(message string, cause error) error {
	return &MaskedError{
		message: message,
		cause:   cause,
	}
}

func WrapOperation(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
