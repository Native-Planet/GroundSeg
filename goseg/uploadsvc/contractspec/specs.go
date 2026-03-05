package contractspec

import (
	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/familyspec"
)

func UploadActionSpecs() []familyspec.ActionSpec {
	return familycatalog.UploadActionFamilySpecs()
}
