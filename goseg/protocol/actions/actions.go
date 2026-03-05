package actions

import (
	"encoding/json"
	"fmt"

	"groundseg/protocol/contracts"
)

// Namespace groups action enums by transport domain.
type Namespace = contracts.ActionNamespace

const (
	NamespaceUpload Namespace = Namespace(contracts.ActionNamespaceUpload)
	NamespaceC2C    Namespace = Namespace(contracts.ActionNamespaceC2C)
)

// Action is a transport action contract token.
type Action string

const (
	ActionUploadOpenEndpoint Action = Action(contracts.ActionUploadOpenEndpoint)
	ActionUploadReset        Action = Action(contracts.ActionUploadReset)
	ActionC2CConnect         Action = Action(contracts.ActionC2CConnect)
)

// UploadPayload indicates which upload payload fragment an action expects.
type UploadPayload = contracts.UploadPayloadRule

const (
	UploadPayloadOpenEndpoint UploadPayload = contracts.UploadPayloadOpenEndpoint
	UploadPayloadReset        UploadPayload = contracts.UploadPayloadReset
)

type actionCatalog struct {
	contractsByNamespace map[Namespace][]ActionContract
	contractByNamespace  map[Namespace]map[Action]ActionContract
	uploadContracts      []UploadActionContract
}

func buildActionCatalog() (actionCatalog, error) {
	if err := contracts.ValidateActionContractBindings(); err != nil {
		return actionCatalog{}, fmt.Errorf("validate action contract bindings: %w", err)
	}
	uploadActions, err := buildActionContractSlice(contracts.ActionNamespace(NamespaceUpload))
	if err != nil {
		return actionCatalog{}, fmt.Errorf("initialize upload action contracts: %w", err)
	}
	c2cActions, err := buildActionContractSlice(contracts.ActionNamespace(NamespaceC2C))
	if err != nil {
		return actionCatalog{}, fmt.Errorf("initialize c2c action contracts: %w", err)
	}
	uploadActionByAction := actionContractByActionIndex(uploadActions)
	c2cActionByAction := actionContractByActionIndex(c2cActions)
	uploadActionContracts, err := buildUploadActionContracts(uploadActions)
	if err != nil {
		return actionCatalog{}, fmt.Errorf("initialize upload action payload matrix: %w", err)
	}
	return actionCatalog{
		contractsByNamespace: map[Namespace][]ActionContract{
			NamespaceUpload: actionContractsFromUpload(uploadActionContracts),
			NamespaceC2C:    c2cActions,
		},
		contractByNamespace: map[Namespace]map[Action]ActionContract{
			NamespaceUpload: uploadActionByAction,
			NamespaceC2C:    c2cActionByAction,
		},
		uploadContracts: uploadActionContracts,
	}, nil
}

// ActionMeta captures protocol action metadata used by protocol-aware dispatchers.
type ActionMeta struct {
	Action      Action
	Description string
}

const C2CPayloadType = "c2c"

type C2CPayload struct {
	Type    string         `json:"type"`
	Payload C2CPayloadBody `json:"payload"`
}

type C2CPayloadBody struct {
	Action   string `json:"action"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type ActionContract struct {
	Action      Action
	ID          contracts.ContractID
	Description string
	Contract    contracts.ContractDescriptor
}

func buildActionContractSlice(namespace contracts.ActionNamespace) ([]ActionContract, error) {
	contractBindings := contracts.ActionContractBindingsForNamespace(namespace)
	contractInfos := make([]ActionContract, 0, len(contractBindings))
	for _, binding := range contractBindings {
		descriptor, ok := contracts.ActionContractFor(namespace, binding.Action)
		if !ok {
			return nil, fmt.Errorf("missing contract descriptor for %s:%s", binding.Namespace, binding.Action)
		}
		contractInfos = append(contractInfos, ActionContract{
			Action:      Action(binding.Action),
			ID:          binding.Contract,
			Description: descriptor.Description,
			Contract:    descriptor,
		})
	}
	return contractInfos, nil
}

func buildUploadActionContracts(uploadActions []ActionContract) ([]UploadActionContract, error) {
	byAction := make(map[Action]ActionContract)
	for _, contract := range uploadActions {
		byAction[contract.Action] = contract
	}
	bindingSpecs := contracts.UploadActionBindingSpecs()
	uploadContracts := make([]UploadActionContract, 0, len(bindingSpecs))
	for _, spec := range bindingSpecs {
		action := Action(spec.Action)
		contract, ok := byAction[action]
		if !ok {
			return nil, fmt.Errorf("missing action contract binding for %q", action)
		}
		uploadContracts = append(uploadContracts, UploadActionContract{
			ActionContract:    contract,
			RequiredPayloads:  UploadPayload(spec.RequiredPayloads),
			ForbiddenPayloads: UploadPayload(spec.ForbiddenPayloads),
		})
		delete(byAction, action)
	}
	if len(byAction) > 0 {
		return nil, fmt.Errorf("unexpected upload action contract bindings: %d", len(byAction))
	}
	return uploadContracts, nil
}

// UploadActionContract defines protocol-level upload action metadata and payload requirements.
type UploadActionContract struct {
	ActionContract
	RequiredPayloads  UploadPayload
	ForbiddenPayloads UploadPayload
}

// currentContractVersion is used by compatibility helpers for explicit policy checks.
const CurrentContractVersion = contracts.CurrentContractVersion

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

// UnsupportedNamespaceError is raised for unknown action namespaces.
type UnsupportedNamespaceError struct {
	Namespace Namespace
}

func (e UnsupportedNamespaceError) Error() string {
	return fmt.Sprintf("unsupported action namespace: %s", e.Namespace)
}

func validateNamespace(namespace Namespace) error {
	switch namespace {
	case NamespaceUpload, NamespaceC2C:
		return nil
	default:
		return UnsupportedNamespaceError{Namespace: namespace}
	}
}

// ParseAction validates an action for a given namespace and returns a typed Action.
func ParseAction(namespace Namespace, raw string) (Action, error) {
	action := Action(raw)
	metas, err := ActionMetas(namespace)
	if err != nil {
		return action, err
	}
	for _, meta := range metas {
		if action == meta.Action {
			return action, nil
		}
	}
	return action, UnsupportedActionError{Namespace: namespace, Action: action}
}

// SupportedActions returns supported action tokens for a namespace.
func SupportedActions(namespace Namespace) ([]Action, error) {
	metas, err := ActionMetas(namespace)
	if err != nil {
		return nil, err
	}
	out := make([]Action, len(metas))
	for i, meta := range metas {
		out[i] = meta.Action
	}
	return out, nil
}

// ActionMetas returns action metadata for a namespace.
func ActionMetas(namespace Namespace) ([]ActionMeta, error) {
	metas, err := actionMetasForNamespace(namespace)
	if err != nil {
		return nil, err
	}
	out := make([]ActionMeta, len(metas))
	copy(out, metas)
	return out, nil
}

func UploadActionContracts() ([]UploadActionContract, error) {
	catalog, err := buildActionCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]UploadActionContract, len(catalog.uploadContracts))
	copy(out, catalog.uploadContracts)
	return out, nil
}

// UploadActionContractByAction returns upload action contracts keyed by action token.
//
// The result is a defensive copy to preserve the initialization order in this package.
func UploadActionContractByAction() (map[Action]UploadActionContract, error) {
	contractsForUpload, err := UploadActionContracts()
	if err != nil {
		return nil, err
	}
	contractsByAction := make(map[Action]UploadActionContract, len(contractsForUpload))
	for _, contract := range contractsForUpload {
		contractsByAction[contract.Action] = contract
	}
	return contractsByAction, nil
}

func actionContractForAction(namespace Namespace, action Action) (ActionContract, error) {
	if err := validateNamespace(namespace); err != nil {
		return ActionContract{}, err
	}
	catalog, err := buildActionCatalog()
	if err != nil {
		return ActionContract{}, err
	}
	contract, ok := catalog.contractByNamespace[namespace][action]
	if !ok {
		return ActionContract{}, UnsupportedActionError{Namespace: namespace, Action: action}
	}
	return contract, nil
}

// ContractForAction returns a contract descriptor for a namespace/action pair.
func ContractForAction(namespace Namespace, action Action) (ActionContract, error) {
	return actionContractForAction(namespace, action)
}

func ContractMetadataForAction(namespace Namespace, action Action) (contracts.ContractMetadata, error) {
	contract, err := actionContractForAction(namespace, action)
	if err != nil {
		return contracts.ContractMetadata{}, err
	}
	return contract.Contract.ContractMetadata, nil
}

func IsActionContractDeprecated(namespace Namespace, version string, action Action) (bool, error) {
	contract, err := actionContractForAction(namespace, action)
	if err != nil {
		return false, err
	}
	return contract.Contract.IsDeprecated(version), nil
}

func IsActionContractActive(namespace Namespace, version string, action Action) (bool, error) {
	contract, err := actionContractForAction(namespace, action)
	if err != nil {
		return false, err
	}
	return contract.Contract.IsActive(version), nil
}

func UploadActionContractForAction(action Action) (UploadActionContract, error) {
	catalog, err := buildActionCatalog()
	if err != nil {
		return UploadActionContract{}, err
	}
	for _, contract := range catalog.uploadContracts {
		if contract.Action == action {
			return contract, nil
		}
	}
	return UploadActionContract{}, UnsupportedActionError{Namespace: NamespaceUpload, Action: action}
}

func actionMetasForNamespace(namespace Namespace) ([]ActionMeta, error) {
	contractsForNamespace, err := actionContractsForNamespace(namespace)
	if err != nil {
		return nil, err
	}
	out := make([]ActionMeta, len(contractsForNamespace))
	for i, contract := range contractsForNamespace {
		out[i] = ActionMeta{
			Action:      contract.Action,
			Description: contract.Description,
		}
	}
	return out, nil
}

func actionContractsForNamespace(namespace Namespace) ([]ActionContract, error) {
	if err := validateNamespace(namespace); err != nil {
		return nil, err
	}
	catalog, err := buildActionCatalog()
	if err != nil {
		return nil, err
	}
	contractsForNamespace := catalog.contractsByNamespace[namespace]
	out := make([]ActionContract, len(contractsForNamespace))
	copy(out, contractsForNamespace)
	return out, nil
}

func actionContractByActionIndex(contractList []ActionContract) map[Action]ActionContract {
	index := make(map[Action]ActionContract, len(contractList))
	for _, contract := range contractList {
		index[contract.Action] = contract
	}
	return index
}

func ActionContracts(namespace Namespace) ([]ActionContract, error) {
	return actionContractsForNamespace(namespace)
}

func NewC2CPayload(action Action, ssid, password string) C2CPayload {
	return C2CPayload{
		Type: C2CPayloadType,
		Payload: C2CPayloadBody{
			Action:   string(action),
			SSID:     ssid,
			Password: password,
		},
	}
}

func MarshalC2CPayload(action Action, ssid, password string) ([]byte, error) {
	payload := NewC2CPayload(action, ssid, password)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal c2c payload for action %s: %w", action, err)
	}
	return encoded, nil
}
