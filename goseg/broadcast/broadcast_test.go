//go:build integration

package broadcast

import (
	"testing"
	"time"
)

func TestUpdateScheduledPackRoundTrip(t *testing.T) {
	patp := "~zod-test"
	want := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)

	if err := DefaultBroadcastStateRuntime().UpdateScheduledPack(patp, want); err != nil {
		t.Fatalf("UpdateScheduledPack returned error: %v", err)
	}

	got := DefaultBroadcastStateRuntime().GetScheduledPack(patp).UTC().Truncate(time.Second)
	if !got.Equal(want) {
		t.Fatalf("unexpected scheduled pack time: want %v got %v", want, got)
	}
}

func TestGetScheduledPackReturnsZeroForMissingPier(t *testing.T) {
	got := DefaultBroadcastStateRuntime().GetScheduledPack("~missing")
	if !got.IsZero() {
		t.Fatalf("expected zero time for missing pier, got %v", got)
	}
}
