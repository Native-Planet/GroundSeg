package routines

import (
	"testing"
	"time"
)

func TestReserveScheduledPackPreventsDuplicateUntilRelease(t *testing.T) {
	scheduledPackState.Lock()
	scheduledPackState.active = make(map[string]time.Time)
	scheduledPackState.Unlock()

	when := time.Unix(100, 0)
	if !reserveScheduledPack("zod", when) {
		t.Fatal("expected first reservation to succeed")
	}
	if reserveScheduledPack("zod", when.Add(time.Minute)) {
		t.Fatal("expected duplicate reservation to be rejected")
	}

	releaseScheduledPack("zod")
	if !reserveScheduledPack("zod", when.Add(time.Minute)) {
		t.Fatal("expected reservation after release to succeed")
	}
	releaseScheduledPack("zod")
}
