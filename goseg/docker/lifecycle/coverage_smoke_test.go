package lifecycle_test

import (
	"testing"

	pkg "groundseg/docker/lifecycle"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewContainerStatusIndex
}
