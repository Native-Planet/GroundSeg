package contracts

func contractMetadataFor(id ContractID) ContractMetadata {
	switch id {
	case UploadOpenEndpointAction, UploadResetAction, C2CConnectAction:
		return ContractMetadata{
			IntroducedIn:  "2026.03.02",
			Compatibility: CompatibilityBackwardSafe,
		}
	case APIConnectionError:
		return ContractMetadata{
			IntroducedIn:  "2026.01.20",
			Compatibility: CompatibilityBackwardSafe,
		}
	default:
		panic("missing canonical contract metadata for " + string(id))
	}
}
