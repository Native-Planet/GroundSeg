package backup_test

import (
	"testing"

	pkg "groundseg/startram/backup"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.GetBackup
}
