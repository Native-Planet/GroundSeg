package startram

import (
	"groundseg/protocol/contracts/catalog/common"
	"groundseg/protocol/contracts/familyspec"
)

const (
	StartramContractFamily = "startram"
	StartramContractScope  = "errors"
	StartramAPIConnSlug    = "api-connection"

	OwnerStartram common.OwnerModule = common.OwnerStartram

	StartramAPIConnectionErrorID          = StartramContractFamily + "." + StartramContractScope + "." + StartramAPIConnSlug
	StartramAPIConnectionErrorMessage     = "Unable to connect to API server"
	StartramAPIConnectionErrorName        = "APIConnectionError"
	StartramAPIConnectionErrorDescription = "Masks transport detail when the StarTram API is unavailable or unreachable."
)

func ContractSpecs() []familyspec.ContractSpec {
	return []familyspec.ContractSpec{
		{
			ID:          StartramAPIConnectionErrorID,
			Name:        StartramAPIConnectionErrorName,
			Description: StartramAPIConnectionErrorDescription,
			Message:     StartramAPIConnectionErrorMessage,
			Owner:       string(OwnerStartram),
		},
	}
}
