package contracts

import (
	"fmt"

	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/startramfamily"
)

const (
	// StarTram contract namespace segments.
	startramContractFamily = familycatalog.StartramContractFamily
	startramContractScope  = familycatalog.StartramContractScope

	// APIConnectionErrorMessage is the canonical StarTram API connectivity contract message.
	APIConnectionErrorMessage             = familycatalog.StartramAPIConnectionErrorMessage
	startramAPIConnectionErrorName        = familycatalog.StartramAPIConnectionErrorName
	startramAPIConnectionErrorDescription = familycatalog.StartramAPIConnectionErrorDescription
)

// APIConnectionError is the canonical StarTram API connectivity contract id.
const APIConnectionError ContractID = ContractID(familycatalog.StartramAPIConnectionErrorID)

func startramContractSpecs() []contractFamilyEntrySpec {
	familySpecs := startramfamily.ContractSpecs()
	specs := make([]contractFamilyEntrySpec, 0, len(familySpecs))
	for _, spec := range familySpecs {
		contractID := ContractID(spec.ID)
		specs = append(specs, contractFamilyEntrySpec{
			ID:          contractID,
			Name:        spec.Name,
			Description: spec.Description,
			Message:     spec.Message,
			Governance:  governanceForContract(contractID, spec.Owner),
		})
	}
	return specs
}

func startramContractCatalogSpecs() []contractCatalogEntry {
	return buildContractFamilyEntries(startramContractSpecs())
}

func validateStartramContractSpecs(entries []contractCatalogEntry) error {
	specs := startramContractSpecs()
	expectedIDs := expectedIDSetFromSpecs(specs)
	return validateContractFamilySpecs(entries, contractFamilyValidationSpec{
		Family:               contractCatalogFamilyStartram,
		IDPrefix:             startramContractFamily + "." + startramContractScope + ".",
		RequireActionBinding: false,
		ExpectedIDs:          expectedIDs,
		ValidateEntry: func(entry contractCatalogEntry) error {
			if entry.ID == "" {
				return fmt.Errorf("startram contract id is required")
			}
			return nil
		},
	})
}
