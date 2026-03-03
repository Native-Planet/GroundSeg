package testutil_test

import (
	"testing"

	pkg "groundseg/testutil"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.WaitForCondition
}
