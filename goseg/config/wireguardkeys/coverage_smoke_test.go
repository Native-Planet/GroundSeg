package wireguardkeys_test

import (
	"testing"

	pkg "groundseg/config/wireguardkeys"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.GenerateKeyPair
}
