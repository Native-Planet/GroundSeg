package shipstatus

import (
	"errors"
	"fmt"
)

var ErrShipStatusNotFound = errors.New("ship status doesn't exist")

func NotFoundErr(patp string) error {
	return fmt.Errorf("%w: %s", ErrShipStatusNotFound, patp)
}
