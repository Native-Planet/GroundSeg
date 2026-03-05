package governance

import "groundseg/protocol/contracts/catalog/common"

const (
	ActionUploadOpenEndpoint = "open-endpoint"
	ActionUploadReset        = "reset"

	UploadOpenEndpointContractID = ProtocolActionContractRoot + "." + NamespaceUpload + "." + ActionUploadOpenEndpoint
	UploadResetContractID        = ProtocolActionContractRoot + "." + NamespaceUpload + "." + ActionUploadReset

	UploadOpenEndpointActionName        = "UploadOpenEndpointAction"
	UploadOpenEndpointActionDescription = "open upload endpoint"
	UploadResetActionName               = "UploadResetAction"
	UploadResetActionDescription        = "reset upload session"
)

func uploadActionDeclarations() []ActionDeclaration {
	return []ActionDeclaration{
		{
			Namespace:         NamespaceUpload,
			Action:            ActionUploadOpenEndpoint,
			ContractID:        UploadOpenEndpointContractID,
			Name:              UploadOpenEndpointActionName,
			Description:       UploadOpenEndpointActionDescription,
			Owner:             string(common.OwnerUploadService),
			RequiredPayloads:  UploadPayloadRuleOpenEndpoint,
			ForbiddenPayloads: UploadPayloadRuleReset,
		},
		{
			Namespace:         NamespaceUpload,
			Action:            ActionUploadReset,
			ContractID:        UploadResetContractID,
			Name:              UploadResetActionName,
			Description:       UploadResetActionDescription,
			Owner:             string(common.OwnerUploadService),
			RequiredPayloads:  UploadPayloadRuleReset,
			ForbiddenPayloads: UploadPayloadRuleOpenEndpoint,
		},
	}
}
