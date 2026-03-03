package config_test

import (
	"testing"

	pkg "groundseg/config"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.RuntimeContextSnapshot
}
