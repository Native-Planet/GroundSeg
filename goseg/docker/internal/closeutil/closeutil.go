package closeutil

import (
	"errors"
	"fmt"
	"reflect"
)

// MergeCloseError closes a Docker client-like resource and merges close errors
// into the operation error accumulator when provided.
func MergeCloseError(closer interface{ Close() error }, operation string, err *error) {
	if closer == nil {
		return
	}
	value := reflect.ValueOf(closer)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return
	}
	closeErr := closer.Close()
	if closeErr == nil {
		return
	}
	closeErr = fmt.Errorf("close docker client during %s: %w", operation, closeErr)
	if err == nil {
		return
	}
	if *err == nil {
		*err = closeErr
		return
	}
	*err = errors.Join(*err, closeErr)
}
