package broadcast

import (
	"testing"

	"groundseg/structs"
)

func TestBroadcastStateStoreOperations(t *testing.T) {
	runtime := NewBroadcastStateRuntime()
	state := structs.AuthBroadcast{}
	state.System.Transition.Error = []string{"existing"}
	if err := runtime.UpdateBroadcast(state); err != nil {
		t.Fatalf("UpdateBroadcast returned error: %v", err)
	}

	if err := runtime.AddSystemTransitionError("latest"); err != nil {
		t.Fatalf("AddSystemTransitionError returned error: %v", err)
	}
	got := runtime.GetState()
	if len(got.System.Transition.Error) == 0 || got.System.Transition.Error[0] != "latest" {
		t.Fatalf("expected latest system transition error to be prepended, got %#v", got.System.Transition.Error)
	}
}
