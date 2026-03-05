package familycatalog

import (
	startramcatalog "groundseg/protocol/contracts/catalog/startram"
	"groundseg/protocol/contracts/familyspec"
)

const (
	StartramContractFamily = startramcatalog.StartramContractFamily
	StartramContractScope  = startramcatalog.StartramContractScope
	StartramAPIConnSlug    = startramcatalog.StartramAPIConnSlug

	StartramAPIConnectionErrorID          = startramcatalog.StartramAPIConnectionErrorID
	StartramAPIConnectionErrorMessage     = startramcatalog.StartramAPIConnectionErrorMessage
	StartramAPIConnectionErrorName        = startramcatalog.StartramAPIConnectionErrorName
	StartramAPIConnectionErrorDescription = startramcatalog.StartramAPIConnectionErrorDescription

	OwnerStartram OwnerModule = OwnerModule(startramcatalog.OwnerStartram)
)

func StartramContractSpecs() []familyspec.ContractSpec {
	return startramcatalog.ContractSpecs()
}
