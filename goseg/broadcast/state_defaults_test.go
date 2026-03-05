package broadcast

import "testing"

func TestSetDefaultBroadcastStateRuntimeUsesProvidedRuntime(t *testing.T) {
	original := DefaultBroadcastStateRuntime()
	t.Cleanup(func() {
		SetDefaultBroadcastStateRuntime(original)
	})

	custom := NewBroadcastStateRuntime()
	if got := SetDefaultBroadcastStateRuntime(custom); got != custom {
		t.Fatalf("expected SetDefaultBroadcastStateRuntime to return provided runtime")
	}
	if got := DefaultBroadcastStateRuntime(); got != custom {
		t.Fatalf("expected default broadcast runtime to match provided runtime")
	}
}

func TestSetDefaultBroadcastStateRuntimeNilCreatesRuntime(t *testing.T) {
	original := DefaultBroadcastStateRuntime()
	t.Cleanup(func() {
		SetDefaultBroadcastStateRuntime(original)
	})

	got := SetDefaultBroadcastStateRuntime(nil)
	if got == nil {
		t.Fatal("expected nil default runtime override to allocate runtime")
	}
	if DefaultBroadcastStateRuntime() != got {
		t.Fatal("expected allocated runtime to become default")
	}
}

func TestResetDefaultBroadcastStateRuntimeReplacesCurrentDefault(t *testing.T) {
	original := DefaultBroadcastStateRuntime()
	t.Cleanup(func() {
		SetDefaultBroadcastStateRuntime(original)
	})

	custom := NewBroadcastStateRuntime()
	SetDefaultBroadcastStateRuntime(custom)
	reset := ResetDefaultBroadcastStateRuntime()
	if reset == nil {
		t.Fatal("expected reset default runtime to be non-nil")
	}
	if reset == custom {
		t.Fatal("expected reset to allocate a distinct runtime instance")
	}
	if DefaultBroadcastStateRuntime() != reset {
		t.Fatal("expected reset runtime to be installed as default")
	}
}
