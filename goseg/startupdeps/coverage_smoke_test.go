package startupdeps_test

import (
	"testing"

	pkg "groundseg/startupdeps"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewStartupDockerRuntime
}
