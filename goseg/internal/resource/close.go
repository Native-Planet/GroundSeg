package resource

import (
	"errors"
	"fmt"
	"io"
)

func JoinCloseError(err error, closer io.Closer, context string) error {
	if closer == nil {
		return err
	}
	closeErr := closer.Close()
	if closeErr == nil {
		return err
	}
	wrapped := fmt.Errorf("%s: %w", context, closeErr)
	if err == nil {
		return wrapped
	}
	return errors.Join(err, wrapped)
}

