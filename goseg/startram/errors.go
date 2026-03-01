package startram

import "groundseg/errpolicy"

const apiConnectionErrorMessage = "Unable to connect to API server"

func wrapAPIConnectionError(err error) error {
	return errpolicy.WrapMasked(apiConnectionErrorMessage, err)
}
