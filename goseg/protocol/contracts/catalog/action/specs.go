package action

import (
	"groundseg/protocol/contracts/catalog/common"
	"groundseg/protocol/contracts/familyspec"
	"groundseg/protocol/contracts/governance"
)

const (
	ProtocolActionContractRoot = governance.ProtocolActionContractRoot

	NamespaceUpload = governance.NamespaceUpload
	NamespaceC2C    = governance.NamespaceC2C

	ActionUploadOpenEndpoint = governance.ActionUploadOpenEndpoint
	ActionUploadReset        = governance.ActionUploadReset
	ActionC2CConnect         = governance.ActionC2CConnect

	UploadOpenEndpointContractID = governance.UploadOpenEndpointContractID
	UploadResetContractID        = governance.UploadResetContractID
	C2CConnectContractID         = governance.C2CConnectContractID

	C2CConnectActionName        = governance.C2CConnectActionName
	C2CConnectActionDescription = governance.C2CConnectActionDescription

	UploadOpenEndpointActionName        = governance.UploadOpenEndpointActionName
	UploadOpenEndpointActionDescription = governance.UploadOpenEndpointActionDescription
	UploadResetActionName               = governance.UploadResetActionName
	UploadResetActionDescription        = governance.UploadResetActionDescription
)

type OwnerModule = common.OwnerModule

const (
	OwnerUploadService = common.OwnerUploadService
	OwnerSystemWiFi    = common.OwnerSystemWiFi
)

type UploadPayloadRule = governance.UploadPayloadRule

const (
	UploadPayloadRuleOpenEndpoint = governance.UploadPayloadRuleOpenEndpoint
	UploadPayloadRuleReset        = governance.UploadPayloadRuleReset
)

type UploadActionSpec struct {
	familyspec.ActionSpec
	RequiredPayloads  UploadPayloadRule
	ForbiddenPayloads UploadPayloadRule
}

func ProtocolActionSpecs() []familyspec.ActionSpec {
	declarations := governance.C2CActionDeclarations()
	out := make([]familyspec.ActionSpec, 0, len(declarations))
	for _, declaration := range declarations {
		out = append(out, familyspec.ActionSpec{
			Namespace:   declaration.Namespace,
			Action:      declaration.Action,
			ContractID:  declaration.ContractID,
			Name:        declaration.Name,
			Description: declaration.Description,
			Owner:       declaration.Owner,
		})
	}
	return out
}

func UploadActionSpecs() []UploadActionSpec {
	declarations := governance.UploadActionDeclarations()
	out := make([]UploadActionSpec, 0, len(declarations))
	for _, declaration := range declarations {
		out = append(out, UploadActionSpec{
			ActionSpec: familyspec.ActionSpec{
				Namespace:   declaration.Namespace,
				Action:      declaration.Action,
				ContractID:  declaration.ContractID,
				Name:        declaration.Name,
				Description: declaration.Description,
				Owner:       declaration.Owner,
			},
			RequiredPayloads:  declaration.RequiredPayloads,
			ForbiddenPayloads: declaration.ForbiddenPayloads,
		})
	}
	return out
}

func UploadActionFamilySpecs() []familyspec.ActionSpec {
	uploadSpecs := UploadActionSpecs()
	out := make([]familyspec.ActionSpec, 0, len(uploadSpecs))
	for _, spec := range uploadSpecs {
		out = append(out, spec.ActionSpec)
	}
	return out
}

func AllActionSpecs() []familyspec.ActionSpec {
	protocolSpecs := ProtocolActionSpecs()
	uploadSpecs := UploadActionFamilySpecs()
	all := make([]familyspec.ActionSpec, 0, len(protocolSpecs)+len(uploadSpecs))
	all = append(all, uploadSpecs...)
	all = append(all, protocolSpecs...)
	return all
}
