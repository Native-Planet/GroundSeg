package orchestration_test

import (
	"testing"

	pkg "groundseg/docker/orchestration"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.ContainerConfigForType
}
