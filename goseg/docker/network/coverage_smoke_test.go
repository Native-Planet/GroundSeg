package network_test

import (
	"testing"

	pkg "groundseg/docker/network"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewNetworkRuntime
}
