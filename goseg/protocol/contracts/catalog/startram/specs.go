package startram

import (
	"groundseg/protocol/contracts/catalog/common"
	"groundseg/protocol/contracts/familyspec"
	"groundseg/protocol/contracts/governance"
)

const (
	StartramContractFamily = governance.StartramContractFamily
	StartramContractScope  = governance.StartramContractScope
	StartramAPIConnSlug    = governance.StartramAPIConnSlug

	OwnerStartram common.OwnerModule = common.OwnerStartram

	StartramAPIConnectionErrorID          = governance.StartramAPIConnectionErrorID
	StartramAPIConnectionErrorMessage     = governance.StartramAPIConnectionErrorMessage
	StartramAPIConnectionErrorName        = governance.StartramAPIConnectionErrorName
	StartramAPIConnectionErrorDescription = governance.StartramAPIConnectionErrorDescription
)

func ContractSpecs() []familyspec.ContractSpec {
	declarations := governance.StartramContractDeclarations()
	out := make([]familyspec.ContractSpec, 0, len(declarations))
	for _, declaration := range declarations {
		out = append(out, familyspec.ContractSpec{
			ID:          declaration.ID,
			Name:        declaration.Name,
			Description: declaration.Description,
			Message:     declaration.Message,
			Owner:       declaration.Owner,
		})
	}
	return out
}
