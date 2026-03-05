package containerstate

import (
	"testing"

	"groundseg/structs"
)

func resetContainerStateForTest(t *testing.T) {
	t.Helper()
	mu.Lock()
	containers = make(map[string]structs.ContainerState)
	mu.Unlock()
}

func TestContainerStateUpdateSnapshotAndDelete(t *testing.T) {
	resetContainerStateForTest(t)
	t.Cleanup(func() {
		resetContainerStateForTest(t)
	})

	initial := structs.ContainerState{Name: "pier", ActualStatus: "Up"}
	Update("pier", initial)

	snapshot := Snapshot()
	got, ok := snapshot["pier"]
	if !ok {
		t.Fatal("expected container to be present in snapshot")
	}
	if got.Name != initial.Name || got.ActualStatus != initial.ActualStatus {
		t.Fatalf("unexpected snapshot value: %#v", got)
	}

	// Ensure Snapshot returns a detached map copy.
	snapshot["pier"] = structs.ContainerState{Name: "mutated", ActualStatus: "Down"}
	if live := Snapshot()["pier"]; live.Name != initial.Name || live.ActualStatus != initial.ActualStatus {
		t.Fatalf("expected snapshot mutation not to affect live state, got %#v", live)
	}

	Delete("pier")
	if _, exists := Snapshot()["pier"]; exists {
		t.Fatal("expected container to be deleted")
	}
}
