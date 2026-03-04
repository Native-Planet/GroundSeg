package contracts

var contractMetadataByID = map[ContractID]ContractMetadata{
	UploadActionOpenEndpoint: {
		IntroducedIn:  "2026.03.02",
		Compatibility: CompatibilityBackwardSafe,
	},
	UploadActionReset: {
		IntroducedIn:  "2026.03.02",
		Compatibility: CompatibilityBackwardSafe,
	},
	C2CConnectAction: {
		IntroducedIn:  "2026.03.02",
		Compatibility: CompatibilityBackwardSafe,
	},
	APIConnectionError: {
		IntroducedIn:  "2026.01.20",
		Compatibility: CompatibilityBackwardSafe,
	},
}

func contractMetadataFor(id ContractID) ContractMetadata {
	metadata, ok := contractMetadataByID[id]
	if !ok {
		panic("missing canonical contract metadata for " + string(id))
	}
	return metadata
}
