package transitionlifecycle_test

import (
	"testing"

	pkg "groundseg/internal/transitionlifecycle"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.LifecyclePlan[string]{}
}
