package familycatalog

import (
	actioncatalog "groundseg/protocol/contracts/catalog/action"
	"groundseg/protocol/contracts/familyspec"
)

const (
	ProtocolActionContractRoot = actioncatalog.ProtocolActionContractRoot

	NamespaceUpload = actioncatalog.NamespaceUpload
	NamespaceC2C    = actioncatalog.NamespaceC2C

	ActionUploadOpenEndpoint = actioncatalog.ActionUploadOpenEndpoint
	ActionUploadReset        = actioncatalog.ActionUploadReset
	ActionC2CConnect         = actioncatalog.ActionC2CConnect

	UploadOpenEndpointContractID = actioncatalog.UploadOpenEndpointContractID
	UploadResetContractID        = actioncatalog.UploadResetContractID
	C2CConnectContractID         = actioncatalog.C2CConnectContractID

	C2CConnectActionName        = actioncatalog.C2CConnectActionName
	C2CConnectActionDescription = actioncatalog.C2CConnectActionDescription

	UploadOpenEndpointActionName        = actioncatalog.UploadOpenEndpointActionName
	UploadOpenEndpointActionDescription = actioncatalog.UploadOpenEndpointActionDescription
	UploadResetActionName               = actioncatalog.UploadResetActionName
	UploadResetActionDescription        = actioncatalog.UploadResetActionDescription
)

type OwnerModule = actioncatalog.OwnerModule

const (
	OwnerUploadService = actioncatalog.OwnerUploadService
	OwnerSystemWiFi    = actioncatalog.OwnerSystemWiFi
)

type UploadPayloadRule = actioncatalog.UploadPayloadRule

const (
	UploadPayloadRuleOpenEndpoint = actioncatalog.UploadPayloadRuleOpenEndpoint
	UploadPayloadRuleReset        = actioncatalog.UploadPayloadRuleReset
)

type UploadActionSpec = actioncatalog.UploadActionSpec

func ProtocolActionSpecs() []familyspec.ActionSpec {
	return actioncatalog.ProtocolActionSpecs()
}

func UploadActionSpecs() []UploadActionSpec {
	return actioncatalog.UploadActionSpecs()
}

func UploadActionFamilySpecs() []familyspec.ActionSpec {
	return actioncatalog.UploadActionFamilySpecs()
}

func AllActionSpecs() []familyspec.ActionSpec {
	return actioncatalog.AllActionSpecs()
}
