package collectors_test

import (
	"testing"

	pkg "groundseg/broadcast/collectors"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.ConstructAppsInfo
}
