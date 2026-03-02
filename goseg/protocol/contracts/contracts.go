package contracts

import "time"

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

func MustContractDescriptor(name ContractID) ContractDescriptor {
	contract, ok := ContractDescriptorFor(name)
	if !ok {
		return ContractDescriptor{Name: string(name)}
	}
	return contract
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
