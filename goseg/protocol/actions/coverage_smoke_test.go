package actions_test

import (
	"testing"

	pkg "groundseg/protocol/actions"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.ParseAction
}
