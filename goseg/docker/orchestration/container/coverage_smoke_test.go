package container_test

import (
	"testing"

	pkg "groundseg/docker/orchestration/container"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.RunContainerWithRuntime
}
