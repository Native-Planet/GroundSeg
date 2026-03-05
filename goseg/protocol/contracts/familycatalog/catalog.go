package familycatalog

import "groundseg/protocol/contracts/familyspec"

const (
	ProtocolActionContractRoot = "protocol.actions"

	NamespaceUpload = "upload"
	NamespaceC2C    = "c2c"

	ActionUploadOpenEndpoint = "open-endpoint"
	ActionUploadReset        = "reset"
	ActionC2CConnect         = "connect"

	UploadOpenEndpointContractID = ProtocolActionContractRoot + "." + NamespaceUpload + "." + ActionUploadOpenEndpoint
	UploadResetContractID        = ProtocolActionContractRoot + "." + NamespaceUpload + "." + ActionUploadReset
	C2CConnectContractID         = ProtocolActionContractRoot + "." + NamespaceC2C + "." + ActionC2CConnect

	C2CConnectActionName        = "C2CConnectAction"
	C2CConnectActionDescription = "connect c2c client"

	UploadOpenEndpointActionName        = "UploadOpenEndpointAction"
	UploadOpenEndpointActionDescription = "open upload endpoint"
	UploadResetActionName               = "UploadResetAction"
	UploadResetActionDescription        = "reset upload session"

	StartramContractFamily = "startram"
	StartramContractScope  = "errors"
	StartramAPIConnSlug    = "api-connection"

	StartramAPIConnectionErrorID          = StartramContractFamily + "." + StartramContractScope + "." + StartramAPIConnSlug
	StartramAPIConnectionErrorMessage     = "Unable to connect to API server"
	StartramAPIConnectionErrorName        = "APIConnectionError"
	StartramAPIConnectionErrorDescription = "Masks transport detail when the StarTram API is unavailable or unreachable."
)

type OwnerModule string

const (
	OwnerUploadService OwnerModule = "upload-domain"
	OwnerSystemWiFi    OwnerModule = "networking-domain"
	OwnerStartram      OwnerModule = "startram-domain"
)

type UploadPayloadRule uint8

const (
	UploadPayloadRuleOpenEndpoint UploadPayloadRule = 1 << iota
	UploadPayloadRuleReset
)

type UploadActionSpec struct {
	familyspec.ActionSpec
	RequiredPayloads  UploadPayloadRule
	ForbiddenPayloads UploadPayloadRule
}

func ProtocolActionSpecs() []familyspec.ActionSpec {
	return []familyspec.ActionSpec{
		{
			Namespace:   NamespaceC2C,
			Action:      ActionC2CConnect,
			ContractID:  C2CConnectContractID,
			Name:        C2CConnectActionName,
			Description: C2CConnectActionDescription,
			Owner:       string(OwnerSystemWiFi),
		},
	}
}

func UploadActionSpecs() []UploadActionSpec {
	return []UploadActionSpec{
		{
			ActionSpec: familyspec.ActionSpec{
				Namespace:   NamespaceUpload,
				Action:      ActionUploadOpenEndpoint,
				ContractID:  UploadOpenEndpointContractID,
				Name:        UploadOpenEndpointActionName,
				Description: UploadOpenEndpointActionDescription,
				Owner:       string(OwnerUploadService),
			},
			RequiredPayloads:  UploadPayloadRuleOpenEndpoint,
			ForbiddenPayloads: UploadPayloadRuleReset,
		},
		{
			ActionSpec: familyspec.ActionSpec{
				Namespace:   NamespaceUpload,
				Action:      ActionUploadReset,
				ContractID:  UploadResetContractID,
				Name:        UploadResetActionName,
				Description: UploadResetActionDescription,
				Owner:       string(OwnerUploadService),
			},
			RequiredPayloads:  UploadPayloadRuleReset,
			ForbiddenPayloads: UploadPayloadRuleOpenEndpoint,
		},
	}
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

func StartramContractSpecs() []familyspec.ContractSpec {
	return []familyspec.ContractSpec{
		{
			ID:          StartramAPIConnectionErrorID,
			Name:        StartramAPIConnectionErrorName,
			Description: StartramAPIConnectionErrorDescription,
			Message:     StartramAPIConnectionErrorMessage,
			Owner:       string(OwnerStartram),
		},
	}
}
