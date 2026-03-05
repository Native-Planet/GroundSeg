package governance

import "groundseg/protocol/contracts/catalog/common"

const (
	StartramContractFamily = "startram"
	StartramContractScope  = "errors"
	StartramAPIConnSlug    = "api-connection"

	StartramAPIConnectionErrorID          = StartramContractFamily + "." + StartramContractScope + "." + StartramAPIConnSlug
	StartramAPIConnectionErrorMessage     = "Unable to connect to API server"
	StartramAPIConnectionErrorName        = "APIConnectionError"
	StartramAPIConnectionErrorDescription = "Masks transport detail when the StarTram API is unavailable or unreachable."
)

func startramContractDeclarations() []ContractDeclaration {
	return []ContractDeclaration{
		{
			ID:          StartramAPIConnectionErrorID,
			Name:        StartramAPIConnectionErrorName,
			Description: StartramAPIConnectionErrorDescription,
			Message:     StartramAPIConnectionErrorMessage,
			Owner:       string(common.OwnerStartram),
		},
	}
}
