package leak_test

import (
	"testing"

	pkg "groundseg/leak"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.GetLickStatuses
}
