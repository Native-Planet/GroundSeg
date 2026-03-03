package system_test

import (
	"testing"

	pkg "groundseg/routines/system"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.StartBackupRoutines
}
