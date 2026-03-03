package broadcast_test

import (
	"testing"

	pkg "groundseg/broadcast"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.BroadcastToClients
}
