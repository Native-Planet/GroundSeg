package shipcleanup_test

import (
	"testing"

	pkg "groundseg/shipcleanup"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.RollbackProvisioning
}
