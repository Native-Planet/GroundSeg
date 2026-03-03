package httpx_test

import (
	"testing"

	pkg "groundseg/httpx"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.ReadBody
}
