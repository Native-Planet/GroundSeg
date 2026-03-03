package api_test

import (
	"testing"

	pkg "groundseg/handler/api"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewShipHandler
}
