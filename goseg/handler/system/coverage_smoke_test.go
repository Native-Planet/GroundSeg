package system_test

import (
	"testing"

	pkg "groundseg/handler/system"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.RecoverWireguardFleet
}
