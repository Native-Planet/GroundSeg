package contracts

const (
	protocolUploadOpenEndpointContractID    = ContractID("protocol.actions.upload.open-endpoint")
	protocolUploadOpenEndpointNamespace     = ActionNamespace("upload")
	protocolUploadOpenEndpointAction        = ActionToken("open-endpoint")
	protocolUploadResetContractID           = ContractID("protocol.actions.upload.reset")
	protocolUploadResetAction               = ActionToken("reset")
	protocolC2CConnectContractID            = ContractID("protocol.actions.c2c.connect")
	protocolC2CConnectNamespace             = ActionNamespace("c2c")
	protocolC2CConnectAction                = ActionToken("connect")
	protocolUploadOpenEndpointIntroducedIn  = "2026.03.02"
	protocolUploadResetIntroducedIn         = "2026.03.02"
	protocolC2CConnectIntroducedIn          = "2026.03.02"
	protocolUploadOpenEndpointCompatibility = CompatibilityBackwardSafe
	protocolUploadResetCompatibility        = CompatibilityBackwardSafe
	protocolC2CConnectCompatibility         = CompatibilityBackwardSafe
)

var (
	protocolUploadOpenEndpointContract = contractCatalogEntry{
		ID:        protocolUploadOpenEndpointContractID,
		Namespace: protocolUploadOpenEndpointNamespace,
		Action:    protocolUploadOpenEndpointAction,
		Descriptor: ContractDescriptor{
			Name:        "UploadActionOpenEndpoint",
			Description: "open upload endpoint",
			ContractMetadata: ContractMetadata{
				IntroducedIn:  protocolUploadOpenEndpointIntroducedIn,
				Compatibility: protocolUploadOpenEndpointCompatibility,
			},
		},
	}

	protocolUploadResetContract = contractCatalogEntry{
		ID:        protocolUploadResetContractID,
		Namespace: protocolUploadOpenEndpointNamespace,
		Action:    protocolUploadResetAction,
		Descriptor: ContractDescriptor{
			Name:        "UploadActionReset",
			Description: "reset upload session",
			ContractMetadata: ContractMetadata{
				IntroducedIn:  protocolUploadResetIntroducedIn,
				Compatibility: protocolUploadResetCompatibility,
			},
		},
	}

	protocolC2CConnectContract = contractCatalogEntry{
		ID:        protocolC2CConnectContractID,
		Namespace: protocolC2CConnectNamespace,
		Action:    protocolC2CConnectAction,
		Descriptor: ContractDescriptor{
			Name:        "C2CConnectAction",
			Description: "connect c2c client",
			ContractMetadata: ContractMetadata{
				IntroducedIn:  protocolC2CConnectIntroducedIn,
				Compatibility: protocolC2CConnectCompatibility,
			},
		},
	}
)

const (
	UploadActionOpenEndpoint ContractID      = protocolUploadOpenEndpointContractID
	UploadActionReset        ContractID      = protocolUploadResetContractID
	C2CConnectAction         ContractID      = protocolC2CConnectContractID
	ActionUploadOpenEndpoint ActionToken     = protocolUploadOpenEndpointAction
	ActionUploadReset        ActionToken     = protocolUploadResetAction
	ActionC2CConnect         ActionToken     = protocolC2CConnectAction
	ActionNamespaceUpload    ActionNamespace = protocolUploadOpenEndpointNamespace
	ActionNamespaceC2C       ActionNamespace = protocolC2CConnectNamespace

	uploadActionOpenEndpointIntroducedIn = protocolUploadOpenEndpointIntroducedIn
	uploadActionResetIntroducedIn        = protocolUploadResetIntroducedIn
	c2cConnectActionIntroducedIn         = protocolC2CConnectIntroducedIn

	uploadActionOpenEndpointCompatibility = protocolUploadOpenEndpointCompatibility
	uploadActionResetCompatibility        = protocolUploadResetCompatibility
	c2cConnectActionCompatibility         = protocolC2CConnectCompatibility
)

var protocolContractCatalogSpecs = []contractCatalogEntry{
	protocolUploadOpenEndpointContract,
	protocolUploadResetContract,
	protocolC2CConnectContract,
}

func protocolContractCatalogEntries() []contractCatalogEntry {
	return catalogEntries(protocolContractCatalogSpecs)
}

func protocolContractCatalogEntriesSnapshot() []contractCatalogEntry {
	return catalogEntriesSnapshot(protocolContractCatalogSpecs)
}
