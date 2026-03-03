package shipworkflow_test

import (
	"testing"

	pkg "groundseg/shipworkflow"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewWireguardRecoveryRuntime
}
