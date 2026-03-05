package broadcast

import (
	"testing"
	"time"
)

func TestBroadcastStateScheduleQueueOperations(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	nextPack := time.Now().Add(time.Minute).Truncate(time.Second)

	if err := runtime.UpdateScheduledPack("~zod", nextPack); err != nil {
		t.Fatalf("UpdateScheduledPack returned error: %v", err)
	}
	if got := runtime.GetScheduledPack("~zod"); !got.Equal(nextPack) {
		t.Fatalf("unexpected scheduled pack value: %v", got)
	}

	if err := runtime.PublishSchedulePack("tick"); err != nil {
		t.Fatalf("PublishSchedulePack returned error: %v", err)
	}
	select {
	case reason := <-runtime.SchedulePackEvents():
		if reason != "tick" {
			t.Fatalf("unexpected schedule reason: %s", reason)
		}
	default:
		t.Fatal("expected publish reason on schedule queue")
	}
}
