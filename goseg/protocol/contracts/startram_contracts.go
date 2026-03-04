package contracts

const (
	// APIConnectionError is the canonical StarTram API connectivity contract id.
	APIConnectionError                    ContractID = "startram.errors.api-connection"
	startramAPIConnectionErrorName                   = "APIConnectionError"
	startramAPIConnectionErrorDescription            = "Masks transport detail when the StarTram API is unavailable or unreachable."
	startramAPIConnectionErrorMessage                = "Unable to connect to API server"
)

func startramContractCatalogSpecs() []contractCatalogEntry {
	return []contractCatalogEntry{
		{
			ID: APIConnectionError,
			Descriptor: ContractDescriptor{
				Name:             startramAPIConnectionErrorName,
				Description:      startramAPIConnectionErrorDescription,
				ContractMetadata: contractMetadataFor(APIConnectionError),
				Message:          startramAPIConnectionErrorMessage,
			},
		},
	}
}
