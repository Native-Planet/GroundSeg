package startram

import "groundseg/errpolicy"

// apiConnectionErrorMessage is the stable public contract for API transport failures.
// The wrapped error must keep the original cause for observability while masking
// transport details from callers.
const apiConnectionErrorMessage = "Unable to connect to API server"

func wrapAPIConnectionError(err error) error {
	return errpolicy.WrapMasked(apiConnectionErrorMessage, err)
}
