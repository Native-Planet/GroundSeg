package transport_test

import (
	"testing"

	pkg "groundseg/system/wifi/transport"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewCaptiveTransportAdapter
}
