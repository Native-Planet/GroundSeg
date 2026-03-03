package events_test

import (
	"testing"

	pkg "groundseg/docker/events"
)

func TestCoverageImportSmoke(t *testing.T) {
	_ = pkg.NewEventRuntime
}
