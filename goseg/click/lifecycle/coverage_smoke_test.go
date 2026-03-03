package lifecycle_test

import (
	"testing"

	pkg "groundseg/click/lifecycle"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.BarExit
}
