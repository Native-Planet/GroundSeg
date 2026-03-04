package contracts

const (
	// Protocol namespaces for websocket action contracts.
	ActionNamespaceUpload ActionNamespace = "upload"
	ActionNamespaceC2C    ActionNamespace = "c2c"

	// Protocol action contract identities (protocol namespace + action token).
	UploadActionOpenEndpoint ContractID  = "protocol.actions.upload.open-endpoint"
	UploadActionReset        ContractID  = "protocol.actions.upload.reset"
	C2CConnectAction         ContractID  = "protocol.actions.c2c.connect"
	ActionUploadOpenEndpoint ActionToken = "open-endpoint"
	ActionUploadReset        ActionToken = "reset"
	ActionC2CConnect         ActionToken = "connect"
)

func protocolContractCatalogSpecs() []contractCatalogEntry {
	return []contractCatalogEntry{
		{
			ID:        UploadActionOpenEndpoint,
			Namespace: ActionNamespaceUpload,
			Action:    ActionUploadOpenEndpoint,
			Descriptor: ContractDescriptor{
				Name:             "UploadActionOpenEndpoint",
				Description:      "open upload endpoint",
				ContractMetadata: contractMetadataFor(UploadActionOpenEndpoint),
			},
		},
		{
			ID:        UploadActionReset,
			Namespace: ActionNamespaceUpload,
			Action:    ActionUploadReset,
			Descriptor: ContractDescriptor{
				Name:             "UploadActionReset",
				Description:      "reset upload session",
				ContractMetadata: contractMetadataFor(UploadActionReset),
			},
		},
		{
			ID:        C2CConnectAction,
			Namespace: ActionNamespaceC2C,
			Action:    ActionC2CConnect,
			Descriptor: ContractDescriptor{
				Name:             "C2CConnectAction",
				Description:      "connect c2c client",
				ContractMetadata: contractMetadataFor(C2CConnectAction),
			},
		},
	}
}
