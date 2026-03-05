package contracts

import (
	"fmt"

	"groundseg/protocol/contracts/familycatalog"
)

const (
	// Protocol namespaces for websocket action contracts.
	ActionNamespaceUpload ActionNamespace = ActionNamespace(familycatalog.NamespaceUpload)
	ActionNamespaceC2C    ActionNamespace = ActionNamespace(familycatalog.NamespaceC2C)

	// Protocol action tokens.
	ActionUploadOpenEndpoint ActionVerb = ActionVerb(familycatalog.ActionUploadOpenEndpoint)
	ActionUploadReset        ActionVerb = ActionVerb(familycatalog.ActionUploadReset)
	ActionC2CConnect         ActionVerb = ActionVerb(familycatalog.ActionC2CConnect)
)

const (
	protocolActionContractRoot = familycatalog.ProtocolActionContractRoot

	// #nosec G101 -- These values are protocol action names required for contract binding, not secrets.
	// Protocol action contract identities are immutable to preserve contract governance boundaries.
	UploadOpenEndpointAction ContractID = ContractID(familycatalog.UploadOpenEndpointContractID)
	UploadResetAction        ContractID = ContractID(familycatalog.UploadResetContractID)
	C2CConnectAction         ContractID = ContractID(familycatalog.C2CConnectContractID)
)

type protocolActionFamilySpec struct {
	Namespace   ActionNamespace
	Action      ActionVerb
	ID          ContractID
	Name        string
	Description string
	Owner       string
}

type UploadActionBindingSpec struct {
	Action            ActionVerb
	Contract          ContractID
	Name              string
	Description       string
	Owner             string
	RequiredPayloads  UploadPayloadRule
	ForbiddenPayloads UploadPayloadRule
}

type UploadPayloadRule uint8

const (
	UploadPayloadOpenEndpoint UploadPayloadRule = UploadPayloadRule(familycatalog.UploadPayloadRuleOpenEndpoint)
	UploadPayloadReset        UploadPayloadRule = UploadPayloadRule(familycatalog.UploadPayloadRuleReset)
)

func (p UploadPayloadRule) Has(flag UploadPayloadRule) bool {
	return p&flag != 0
}

func (p UploadPayloadRule) IsEmpty() bool {
	return p == 0
}

func newProtocolActionSpec(id ContractID, namespace ActionNamespace, action ActionVerb, name, description, owner string) contractFamilyEntrySpec {
	return contractFamilyEntrySpec{
		ID:          id,
		Namespace:   namespace,
		Action:      action,
		Name:        name,
		Description: description,
		Governance:  governanceForContract(id, owner),
	}
}

func protocolActionFamilySpecs() []protocolActionFamilySpec {
	allSpecs := familycatalog.AllActionSpecs()
	specs := make([]protocolActionFamilySpec, 0, len(allSpecs))
	for _, spec := range allSpecs {
		specs = append(specs, protocolActionFamilySpec{
			Namespace:   ActionNamespace(spec.Namespace),
			Action:      ActionVerb(spec.Action),
			ID:          ContractID(spec.ContractID),
			Name:        spec.Name,
			Description: spec.Description,
			Owner:       spec.Owner,
		})
	}
	return specs
}

func protocolContractSpecs() []contractFamilyEntrySpec {
	familySpecs := protocolActionFamilySpecs()
	specs := make([]contractFamilyEntrySpec, 0, len(familySpecs))
	for _, spec := range familySpecs {
		specs = append(specs, newProtocolActionSpec(
			spec.ID,
			spec.Namespace,
			spec.Action,
			spec.Name,
			spec.Description,
			spec.Owner,
		))
	}
	return specs
}

func UploadActionBindingSpecs() []UploadActionBindingSpec {
	specs := familycatalog.UploadActionSpecs()
	out := make([]UploadActionBindingSpec, 0, len(specs))
	for _, spec := range specs {
		out = append(out, UploadActionBindingSpec{
			Action:            ActionVerb(spec.Action),
			Contract:          ContractID(spec.ContractID),
			Name:              spec.Name,
			Description:       spec.Description,
			Owner:             spec.Owner,
			RequiredPayloads:  UploadPayloadRule(spec.RequiredPayloads),
			ForbiddenPayloads: UploadPayloadRule(spec.ForbiddenPayloads),
		})
	}
	return out
}

func protocolContractCatalogSpecs() []contractCatalogEntry {
	return buildContractFamilyEntries(protocolContractSpecs())
}

func validateProtocolContractSpecs(entries []contractCatalogEntry) error {
	specs := protocolContractSpecs()
	expectedByBinding := actionBindingIDIndexFromSpecs(specs)
	expectedIDs := expectedIDSetFromSpecs(specs)
	return validateContractFamilySpecs(entries, contractFamilyValidationSpec{
		Family:               contractCatalogFamilyProtocol,
		IDPrefix:             protocolActionContractRoot + ".",
		RequireActionBinding: true,
		ExpectedIDs:          expectedIDs,
		ValidateEntry: func(entry contractCatalogEntry) error {
			key := actionContractBindingKey{Namespace: entry.Namespace, Action: entry.Action}
			expectedID, ok := expectedByBinding[key]
			if !ok {
				return fmt.Errorf("unexpected protocol action binding %s:%s", entry.Namespace, entry.Action)
			}
			if entry.ID != expectedID {
				return fmt.Errorf("protocol action %s:%s id mismatch: got %s want %s", entry.Namespace, entry.Action, entry.ID, expectedID)
			}
			if expectedByFormula := protocolActionContractID(entry.Namespace, entry.Action); entry.ID != expectedByFormula {
				return fmt.Errorf("protocol action %s:%s formula mismatch: got %s want %s", entry.Namespace, entry.Action, entry.ID, expectedByFormula)
			}
			return nil
		},
	})
}
