package broadcast

import (
	"errors"
	"testing"

	"groundseg/structs"
)

func TestBroadcastToClientsWithRuntimeRequiresRuntimeSentinel(t *testing.T) {
	if err := broadcastToClientsWithRuntime(nil); err == nil {
		t.Fatal("expected missing runtime error")
	} else if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}

func TestApplyBroadcastTransitionRequiresRuntimeSentinel(t *testing.T) {
	err := ApplyBroadcastTransition(false, broadcastMutationTransition{
		mutate: func(*structs.AuthBroadcast) {},
	}, nil)
	if err == nil {
		t.Fatal("expected missing runtime error")
	}
	if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}
