package seams_test

import (
	"testing"

	pkg "groundseg/internal/seams"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.ErrRuntimeUnconfigured
}
