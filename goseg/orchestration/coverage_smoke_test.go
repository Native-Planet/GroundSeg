package orchestration_test

import (
	"testing"

	pkg "groundseg/orchestration"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewTransitionPolicy
}
