package actions

import (
	"fmt"

	"groundseg/protocol/contracts"
)

// Namespace groups action enums by transport domain.
type Namespace string

const (
	NamespaceUpload Namespace = "upload"
	NamespaceC2C    Namespace = "c2c"
)

// Action is a transport action contract token.
type Action string

const (
	ActionUploadOpenEndpoint Action = "open-endpoint"
	ActionUploadReset        Action = "reset"
	ActionC2CConnect         Action = "connect"
)

// UploadPayload indicates which upload payload fragment an action expects.
type UploadPayload uint8

const (
	UploadPayloadOpenEndpoint UploadPayload = 1 << iota
	UploadPayloadReset
)

var (
	uploadActionContracts = buildUploadActionContracts()
	c2cActionContracts    = buildActionContractSlice(contracts.ActionNamespaceC2C)
)

var actionContractsByNamespace = map[Namespace][]ActionContract{
	NamespaceUpload: actionContractsFromUpload(uploadActionContracts),
	NamespaceC2C:    c2cActionContracts,
}

// ActionMeta captures protocol action metadata used by protocol-aware dispatchers.
type ActionMeta struct {
	Action      Action
	Description string
}

type ActionContract struct {
	Action      Action
	Description string
	Contract    contracts.ContractDescriptor
}

func buildActionContractSlice(namespace contracts.ActionNamespace) []ActionContract {
	contractBindings := contracts.ActionContractBindingsForNamespace(string(namespace))
	contractInfos := make([]ActionContract, 0, len(contractBindings))
	for _, binding := range contractBindings {
		descriptor, ok := contracts.ActionContractDescriptor(string(binding.Namespace), binding.Action)
		if !ok {
			panic(fmt.Sprintf("missing contract descriptor for %s:%s", binding.Namespace, binding.Action))
		}
		contractInfos = append(contractInfos, ActionContract{
			Action:      Action(binding.Action),
			Description: descriptor.Description,
			Contract:    descriptor,
		})
	}
	return contractInfos
}

func buildUploadActionContracts() []UploadActionContract {
	uploadActions := buildActionContractSlice(contracts.ActionNamespaceUpload)
	byAction := make(map[Action]ActionContract)
	for _, contract := range uploadActions {
		byAction[contract.Action] = contract
	}
	return []UploadActionContract{
		{
			ActionContract:    byAction[ActionUploadOpenEndpoint],
			RequiredPayloads:  UploadPayloadOpenEndpoint,
			ForbiddenPayloads: UploadPayloadReset,
		},
		{
			ActionContract:    byAction[ActionUploadReset],
			RequiredPayloads:  UploadPayloadReset,
			ForbiddenPayloads: UploadPayloadOpenEndpoint,
		},
	}
}

// UploadActionContract defines protocol-level upload action metadata and payload requirements.
type UploadActionContract struct {
	ActionContract
	RequiredPayloads  UploadPayload
	ForbiddenPayloads UploadPayload
}

// currentContractVersion is used by compatibility helpers for explicit policy checks.
const CurrentContractVersion = contracts.CurrentContractVersion

func (p UploadPayload) Has(flag UploadPayload) bool {
	return p&flag != 0
}

// IsEmpty reports whether no payload behavior is required or forbidden.
func (p UploadPayload) IsEmpty() bool {
	return p == 0
}

func actionContractsFromUpload(contracts []UploadActionContract) []ActionContract {
	out := make([]ActionContract, len(contracts))
	for i, contract := range contracts {
		out[i] = contract.ActionContract
	}
	return out
}

// UnsupportedActionError is raised for unknown action values within a namespace.
type UnsupportedActionError struct {
	Namespace Namespace
	Action    Action
}

func (e UnsupportedActionError) Error() string {
	return fmt.Sprintf("unsupported %s action: %s", e.Namespace, e.Action)
}

// ParseAction validates an action for a given namespace and returns a typed Action.
func ParseAction(namespace Namespace, raw string) (Action, error) {
	action := Action(raw)
	for _, meta := range actionMetasForNamespace(namespace) {
		if action == meta.Action {
			return action, nil
		}
	}
	return action, UnsupportedActionError{Namespace: namespace, Action: action}
}

// SupportedActions returns supported action tokens for a namespace.
func SupportedActions(namespace Namespace) []Action {
	metas := actionMetasForNamespace(namespace)
	if len(metas) == 0 {
		return nil
	}
	out := make([]Action, len(metas))
	for i, meta := range metas {
		out[i] = meta.Action
	}
	return out
}

// ActionMetas returns action metadata for a namespace.
func ActionMetas(namespace Namespace) []ActionMeta {
	metas := actionMetasForNamespace(namespace)
	if len(metas) == 0 {
		return nil
	}
	out := make([]ActionMeta, len(metas))
	copy(out, metas)
	return out
}

func UploadActionContracts() []UploadActionContract {
	out := make([]UploadActionContract, len(uploadActionContracts))
	copy(out, uploadActionContracts)
	return out
}

func actionContractForAction(namespace Namespace, action Action) (ActionContract, bool) {
	for _, contract := range actionContractsForNamespace(namespace) {
		if contract.Action == action {
			return contract, true
		}
	}
	return ActionContract{}, false
}

func actionContractMetadataForAction(namespace Namespace, action Action) (contracts.ContractMetadata, bool) {
	contract, ok := actionContractForAction(namespace, action)
	if !ok {
		return contracts.ContractMetadata{}, false
	}
	return contract.Contract.ContractMetadata, true
}

func isActionContractDeprecated(namespace Namespace, version string, action Action) bool {
	contract, ok := actionContractForAction(namespace, action)
	if !ok {
		return false
	}
	return contract.Contract.IsDeprecated(version)
}

func isActionContractActive(namespace Namespace, version string, action Action) bool {
	contract, ok := actionContractForAction(namespace, action)
	if !ok {
		return false
	}
	return contract.Contract.IsActive(version)
}

func UploadActionContractForAction(action Action) (UploadActionContract, bool) {
	for _, contract := range uploadActionContracts {
		if contract.Action == action {
			return contract, true
		}
	}
	return UploadActionContract{}, false
}

// UploadActionContractMetadataForAction returns metadata for action compatibility checks.
func UploadActionContractMetadataForAction(action Action) (contracts.ContractMetadata, bool) {
	return actionContractMetadataForAction(NamespaceUpload, action)
}

// IsUploadActionContractDeprecated reports whether an action contract is deprecated as of version.
func IsUploadActionContractDeprecated(version string, action Action) bool {
	return isActionContractDeprecated(NamespaceUpload, version, action)
}

// IsUploadActionContractActive reports whether an action contract is active at the provided version.
func IsUploadActionContractActive(version string, action Action) bool {
	return isActionContractActive(NamespaceUpload, version, action)
}

func actionMetasForNamespace(namespace Namespace) []ActionMeta {
	contracts := actionContractsForNamespace(namespace)
	if len(contracts) == 0 {
		return nil
	}
	out := make([]ActionMeta, len(contracts))
	for i, contract := range contracts {
		out[i] = ActionMeta{
			Action:      contract.Action,
			Description: contract.Description,
		}
	}
	return out
}

func actionContractsForNamespace(namespace Namespace) []ActionContract {
	contractsForNamespace, ok := actionContractsByNamespace[namespace]
	if !ok {
		return nil
	}
	out := make([]ActionContract, len(contractsForNamespace))
	copy(out, contractsForNamespace)
	return out
}

func C2CActionContracts() []ActionContract {
	out := make([]ActionContract, len(c2cActionContracts))
	copy(out, c2cActionContracts)
	return out
}

func C2CActionContractForAction(action Action) (ActionContract, bool) {
	return actionContractForAction(NamespaceC2C, action)
}

// C2CActionContractMetadataForAction returns metadata for C2C action compatibility checks.
func C2CActionContractMetadataForAction(action Action) (contracts.ContractMetadata, bool) {
	return actionContractMetadataForAction(NamespaceC2C, action)
}

// IsC2CActionContractDeprecated reports whether a C2C action contract is deprecated as of version.
func IsC2CActionContractDeprecated(version string, action Action) bool {
	return isActionContractDeprecated(NamespaceC2C, version, action)
}

// IsC2CActionContractActive reports whether a C2C action contract is active at the provided version.
func IsC2CActionContractActive(version string, action Action) bool {
	return isActionContractActive(NamespaceC2C, version, action)
}

// ParseUploadAction validates actions for the upload transport namespace.
func ParseUploadAction(raw string) (Action, error) {
	return ParseAction(NamespaceUpload, raw)
}

// ParseC2CAction validates actions for the c2c transport namespace.
func ParseC2CAction(raw string) (Action, error) {
	return ParseAction(NamespaceC2C, raw)
}

// SupportedUploadActions returns upload-supported actions.
func SupportedUploadActions() []Action {
	return SupportedActions(NamespaceUpload)
}

// SupportedC2CActions returns c2c-supported actions.
func SupportedC2CActions() []Action {
	return SupportedActions(NamespaceC2C)
}
