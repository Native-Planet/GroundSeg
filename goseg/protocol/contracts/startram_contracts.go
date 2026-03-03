package contracts

var (
	apiConnectionErrorMessage = "Unable to connect to API server"
)

var (
	startramApiConnectionErrorContract = contractCatalogEntry{
		ID: ContractID("startram.errors.api-connection"),
		Descriptor: ContractDescriptor{
			Name:        "APIConnectionError",
			Description: "Masks transport detail when the StarTram API is unavailable or unreachable.",
			ContractMetadata: ContractMetadata{
				IntroducedIn:  "2026.01.20",
				Compatibility: CompatibilityBackwardSafe,
			},
			Message: apiConnectionErrorMessage,
		},
	}
)

var (
	APIConnectionError                      ContractID            = startramApiConnectionErrorContract.ID
	startramApiConnectionErrorIntroducedIn                        = startramApiConnectionErrorContract.Descriptor.IntroducedIn
	startramApiConnectionErrorCompatibility ContractCompatibility = startramApiConnectionErrorContract.Descriptor.Compatibility
)

var startramContractCatalogSpecs = []contractCatalogEntry{
	startramApiConnectionErrorContract,
}

func startramContractCatalogEntries() []contractCatalogEntry {
	return append([]contractCatalogEntry(nil), startramContractCatalogSpecs...)
}

func startramContractCatalogEntriesSnapshot() []contractCatalogEntry {
	return append([]contractCatalogEntry(nil), startramContractCatalogEntries()...)
}
