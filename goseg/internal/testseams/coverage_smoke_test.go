package testseams_test

import (
	"testing"

	pkg "groundseg/internal/testseams"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.Override[string]
}
