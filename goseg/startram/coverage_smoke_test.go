package startram_test

import (
	"testing"

	pkg "groundseg/startram"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.SetRegistrationService
}
