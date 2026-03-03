package session_test

import (
	"testing"

	pkg "groundseg/session"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewLogstreamRuntime
}
