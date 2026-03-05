package contracts

import (
	"fmt"
	"strings"
	"time"
)

type contractGovernanceMetadata struct {
	OwnerModule  string
	Stability    ContractCompatibility
	SinceVersion string
	Deprecates   ContractID
}

type contractFamilyEntrySpec struct {
	ID          ContractID
	Namespace   ActionNamespace
	Action      ActionVerb
	Name        string
	Description string
	Message     string
	Governance  contractGovernanceMetadata
}

type contractFamilyValidationSpec struct {
	Family               string
	IDPrefix             string
	RequireActionBinding bool
	ExpectedIDs          map[ContractID]struct{}
	ValidateEntry        func(contractCatalogEntry) error
}

func governanceForContract(id ContractID, owner string) contractGovernanceMetadata {
	metadata := contractMetadataFor(id)
	return contractGovernanceMetadata{
		OwnerModule:  owner,
		Stability:    metadata.Compatibility,
		SinceVersion: metadata.IntroducedIn,
	}
}

func buildContractFamilyEntries(specs []contractFamilyEntrySpec) []contractCatalogEntry {
	entries := make([]contractCatalogEntry, 0, len(specs))
	for _, spec := range specs {
		entries = append(entries, contractCatalogEntry{
			ID:         spec.ID,
			Namespace:  spec.Namespace,
			Action:     spec.Action,
			Governance: spec.Governance,
			Descriptor: ContractDescriptor{
				Name:             spec.Name,
				Description:      spec.Description,
				ContractMetadata: contractMetadataFor(spec.ID),
				Message:          spec.Message,
			},
		})
	}
	return entries
}

func validateContractFamilySpecs(entries []contractCatalogEntry, spec contractFamilyValidationSpec) error {
	if err := validateFamilyCatalogEntries(spec.Family, entries); err != nil {
		return err
	}
	if len(spec.ExpectedIDs) > 0 && len(entries) != len(spec.ExpectedIDs) {
		return fmt.Errorf("expected %d %s contract entries, got %d", len(spec.ExpectedIDs), spec.Family, len(entries))
	}
	for _, entry := range entries {
		if spec.IDPrefix != "" && !strings.HasPrefix(string(entry.ID), spec.IDPrefix) {
			return fmt.Errorf("%s contract %s should use %q prefix", spec.Family, entry.ID, spec.IDPrefix)
		}
		if spec.RequireActionBinding {
			if entry.Namespace == "" || entry.Action == "" {
				return fmt.Errorf("%s contract %s requires namespace/action binding", spec.Family, entry.ID)
			}
		} else if entry.Namespace != "" || entry.Action != "" {
			return fmt.Errorf("%s contract %s should not define namespace/action binding", spec.Family, entry.ID)
		}
		if len(spec.ExpectedIDs) > 0 {
			if _, ok := spec.ExpectedIDs[entry.ID]; !ok {
				return fmt.Errorf("unexpected %s contract id %s", spec.Family, entry.ID)
			}
		}
		if spec.ValidateEntry != nil {
			if err := spec.ValidateEntry(entry); err != nil {
				return fmt.Errorf("validate %s contract %s: %w", spec.Family, entry.ID, err)
			}
		}
		if err := validateCatalogEntryGovernance(entry); err != nil {
			return fmt.Errorf("validate %s contract governance for %s: %w", spec.Family, entry.ID, err)
		}
		if err := validateCatalogEntryGovernanceMetadata(entry, spec.ExpectedIDs); err != nil {
			return fmt.Errorf("validate %s governance metadata for %s: %w", spec.Family, entry.ID, err)
		}
	}
	return nil
}

func validateCatalogEntryGovernance(entry contractCatalogEntry) error {
	if !entry.Descriptor.IsActive(CurrentContractVersion) {
		return fmt.Errorf("contract %s is not active at governance version %s", entry.ID, CurrentContractVersion)
	}
	return nil
}

func validateCatalogEntryGovernanceMetadata(entry contractCatalogEntry, knownIDs map[ContractID]struct{}) error {
	governance := entry.Governance
	if strings.TrimSpace(governance.OwnerModule) == "" {
		return fmt.Errorf("owner module is required")
	}
	if strings.TrimSpace(governance.SinceVersion) == "" {
		return fmt.Errorf("since version is required")
	}
	if _, err := time.Parse(contractVersionLayout, governance.SinceVersion); err != nil {
		return fmt.Errorf("invalid since version %q: %w", governance.SinceVersion, err)
	}
	if !isKnownCompatibility(governance.Stability) {
		return fmt.Errorf("unknown stability %q", governance.Stability)
	}
	introduced := entry.Descriptor.ContractMetadata.IntroducedIn
	if introduced != "" && governance.SinceVersion != introduced {
		return fmt.Errorf("since version %q does not match introduced version %q", governance.SinceVersion, introduced)
	}
	if governance.Deprecates != "" {
		if governance.Deprecates == entry.ID {
			return fmt.Errorf("deprecates cannot point to self")
		}
		if len(knownIDs) > 0 {
			if _, ok := knownIDs[governance.Deprecates]; !ok {
				return fmt.Errorf("deprecates references unknown contract %s", governance.Deprecates)
			}
		}
	}
	return nil
}

func expectedIDSetFromSpecs(specs []contractFamilyEntrySpec) map[ContractID]struct{} {
	expected := make(map[ContractID]struct{}, len(specs))
	for _, spec := range specs {
		expected[spec.ID] = struct{}{}
	}
	return expected
}

func actionBindingIDIndexFromSpecs(specs []contractFamilyEntrySpec) map[actionContractBindingKey]ContractID {
	index := make(map[actionContractBindingKey]ContractID, len(specs))
	for _, spec := range specs {
		if spec.Namespace == "" || spec.Action == "" {
			continue
		}
		index[actionContractBindingKey{
			Namespace: spec.Namespace,
			Action:    spec.Action,
		}] = spec.ID
	}
	return index
}
