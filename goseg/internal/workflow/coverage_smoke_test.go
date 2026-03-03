package workflow_test

import (
	"testing"

	pkg "groundseg/internal/workflow"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.Collect
}
