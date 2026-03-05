package startramfamily

import (
	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/familyspec"
)

func ContractSpecs() []familyspec.ContractSpec {
	return familycatalog.StartramContractSpecs()
}
