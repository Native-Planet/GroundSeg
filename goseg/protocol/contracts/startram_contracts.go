package contracts

const (
	startramApiConnectionErrorContractID                          = ContractID("startram.errors.api-connection")
	startramApiConnectionErrorName                                = "APIConnectionError"
	startramApiConnectionErrorDescription                         = "Masks transport detail when the StarTram API is unavailable or unreachable."
	startramApiConnectionErrorMessage                             = "Unable to connect to API server"
	startramApiConnectionErrorIntroducedIn                        = "2026.01.20"
	startramApiConnectionErrorCompatibility ContractCompatibility = CompatibilityBackwardSafe
)

var (
	startramApiConnectionErrorContract = contractCatalogEntry{
		ID: startramApiConnectionErrorContractID,
		Descriptor: ContractDescriptor{
			Name:        startramApiConnectionErrorName,
			Description: startramApiConnectionErrorDescription,
			ContractMetadata: ContractMetadata{
				IntroducedIn:  startramApiConnectionErrorIntroducedIn,
				Compatibility: startramApiConnectionErrorCompatibility,
			},
			Message: startramApiConnectionErrorMessage,
		},
	}
)

const APIConnectionError ContractID = startramApiConnectionErrorContractID

var startramContractCatalogSpecs = []contractCatalogEntry{
	startramApiConnectionErrorContract,
}

func startramContractCatalogEntries() []contractCatalogEntry {
	return catalogEntries(startramContractCatalogSpecs)
}

func startramContractCatalogEntriesSnapshot() []contractCatalogEntry {
	return catalogEntriesSnapshot(startramContractCatalogSpecs)
}
