package contracts

import (
	"errors"
	"fmt"
	"time"
)

// ContractCompatibility models a contract's compatibility policy across versions.
type ContractCompatibility string

const (
	CompatibilityStable       ContractCompatibility = "stable"
	CompatibilityDeprecated   ContractCompatibility = "deprecated"
	CompatibilityRemoved      ContractCompatibility = "removed"
	CompatibilityBackwardSafe ContractCompatibility = "backward-compatible"
)

// ContractMetadata captures lifecycle and compatibility information for a contract.
type ContractMetadata struct {
	IntroducedIn  string
	DeprecatedIn  string
	RemovedIn     string
	Compatibility ContractCompatibility
}

// ContractID identifies a contract in the shared governance catalog.
type ContractID string

const (
	UploadActionOpenEndpoint ContractID = "protocol.actions.upload.open-endpoint"
	UploadActionReset        ContractID = "protocol.actions.upload.reset"
	C2CConnectAction         ContractID = "protocol.actions.c2c.connect"
	APIConnectionError       ContractID = "startram.errors.api-connection"
)

// ActionNamespace identifies the protocol namespace for action contracts.
type ActionNamespace string

const (
	ActionNamespaceUpload ActionNamespace = "upload"
	ActionNamespaceC2C    ActionNamespace = "c2c"
)

// ActionContractBinding joins an action identifier with its contract descriptor ID
// and namespace, allowing runtime validation and single-source discovery.
type ActionContractBinding struct {
	Namespace ActionNamespace
	Action    string
	Contract  ContractID
}

// ContractDescriptor captures both lifecycle metadata and human-facing contract details.
type ContractDescriptor struct {
	Name        string
	Description string
	ContractMetadata
	Message string
}

func (contract ContractDescriptor) IsDeprecated(version string) bool {
	return IsContractDeprecated(version, contract.ContractMetadata)
}

func (contract ContractDescriptor) IsActive(version string) bool {
	return IsContractActive(version, contract.ContractMetadata)
}

var contractCatalog = map[ContractID]ContractDescriptor{
	UploadActionOpenEndpoint: {
		Name:        "UploadActionOpenEndpoint",
		Description: "open upload endpoint",
		ContractMetadata: ContractMetadata{
			IntroducedIn:  CurrentContractVersion,
			Compatibility: CompatibilityBackwardSafe,
		},
	},
	UploadActionReset: {
		Name:        "UploadActionReset",
		Description: "reset upload session",
		ContractMetadata: ContractMetadata{
			IntroducedIn:  CurrentContractVersion,
			Compatibility: CompatibilityBackwardSafe,
		},
	},
	C2CConnectAction: {
		Name:        "C2CActionConnect",
		Description: "connect c2c client",
		ContractMetadata: ContractMetadata{
			IntroducedIn:  CurrentContractVersion,
			Compatibility: CompatibilityBackwardSafe,
		},
	},
	APIConnectionError: {
		Name:        "APIConnectionError",
		Description: "Masks transport detail when the StarTram API is unavailable or unreachable.",
		ContractMetadata: ContractMetadata{
			IntroducedIn:  CurrentContractVersion,
			Compatibility: CompatibilityBackwardSafe,
		},
		Message: "Unable to connect to API server",
	},
}

func ContractDescriptorFor(name ContractID) (ContractDescriptor, bool) {
	contract, ok := contractCatalog[name]
	return contract, ok
}

var errMissingContractDescriptor = errors.New("contract descriptor lookup failed")

func MustContractDescriptor(name ContractID) ContractDescriptor {
	contract, ok := ContractDescriptorFor(name)
	if !ok {
		panic(fmt.Sprintf("missing contract descriptor for %s", name))
	}
	return contract
}

// ActionContractBindings defines the canonical action-to-contract registry.
var ActionContractBindings = []ActionContractBinding{
	{
		Namespace: ActionNamespaceUpload,
		Action:    "open-endpoint",
		Contract:  UploadActionOpenEndpoint,
	},
	{
		Namespace: ActionNamespaceUpload,
		Action:    "reset",
		Contract:  UploadActionReset,
	},
	{
		Namespace: ActionNamespaceC2C,
		Action:    "connect",
		Contract:  C2CConnectAction,
	},
}

func ActionContractDescriptor(namespace, action string) (ContractDescriptor, bool) {
	for _, binding := range ActionContractBindings {
		if string(binding.Namespace) != namespace || binding.Action != action {
			continue
		}
		contract, ok := ContractDescriptorFor(binding.Contract)
		return contract, ok
	}
	return ContractDescriptor{}, false
}

func ActionContractBindingsForNamespace(namespace string) []ActionContractBinding {
	out := make([]ActionContractBinding, 0, len(ActionContractBindings))
	for _, binding := range ActionContractBindings {
		if string(binding.Namespace) == namespace {
			out = append(out, binding)
		}
	}
	return out
}

func ValidateActionContractBindings() error {
	seen := map[string]struct{}{}
	for _, binding := range ActionContractBindings {
		if binding.Namespace == "" {
			return fmt.Errorf("%w: missing namespace for action %s", errMissingContractDescriptor, binding.Action)
		}
		if binding.Action == "" {
			return fmt.Errorf("%w: missing action in namespace %s", errMissingContractDescriptor, binding.Namespace)
		}
		descriptor, ok := ContractDescriptorFor(binding.Contract)
		if !ok {
			return fmt.Errorf("%w: missing contract %s for action %s:%s", errMissingContractDescriptor, binding.Contract, binding.Namespace, binding.Action)
		}
		if descriptor.Name == "" {
			return fmt.Errorf("%w: contract %s has missing name", errMissingContractDescriptor, binding.Contract)
		}
		if descriptor.Description == "" {
			return fmt.Errorf("%w: contract %s has missing description", errMissingContractDescriptor, binding.Contract)
		}
		if _, seenBefore := seen[string(binding.Namespace)+":"+binding.Action]; seenBefore {
			return fmt.Errorf("%w: duplicate action binding %s:%s", errMissingContractDescriptor, binding.Namespace, binding.Action)
		}
		seen[string(binding.Namespace)+":"+binding.Action] = struct{}{}
	}
	return nil
}

func init() {
	if err := ValidateActionContractBindings(); err != nil {
		panic(err)
	}
}

const contractVersionLayout = "2006.01.02"

// CurrentContractVersion is the canonical active contract version for compatibility checks.
const CurrentContractVersion = "2026.03.02"

// IsVersionAtLeast reports whether version is at or after target.
func IsVersionAtLeast(version string, target string) bool {
	if target == "" {
		return false
	}
	v, err := time.Parse(contractVersionLayout, version)
	if err != nil {
		return false
	}
	cutoff, err := time.Parse(contractVersionLayout, target)
	if err != nil {
		return false
	}
	return !v.Before(cutoff)
}

// IsVersionAtLeastOrEqual is an alias for IsVersionAtLeast for readability.
func IsVersionAtLeastOrEqual(version string, target string) bool {
	return IsVersionAtLeast(version, target)
}

// IsContractDeprecated reports whether the version is within the explicit deprecated
// window and not yet removed.
func IsContractDeprecated(version string, metadata ContractMetadata) bool {
	return metadata.DeprecatedIn != "" &&
		IsVersionAtLeast(version, metadata.DeprecatedIn) &&
		!IsVersionAtLeastOrEqual(version, metadata.RemovedIn)
}

// IsContractActive reports whether the contract is introduced and not yet removed at version.
func IsContractActive(version string, metadata ContractMetadata) bool {
	if !IsVersionAtLeast(version, metadata.IntroducedIn) {
		return false
	}
	return !IsVersionAtLeastOrEqual(version, metadata.RemovedIn)
}
