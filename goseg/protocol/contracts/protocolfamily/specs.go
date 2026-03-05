package protocolfamily

import (
	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/familyspec"
)

func ActionSpecs() []familyspec.ActionSpec {
	return familycatalog.ProtocolActionSpecs()
}
