package subsystem_test

import (
	"testing"

	pkg "groundseg/docker/orchestration/subsystem"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.StartDockerHealthLoops
}
