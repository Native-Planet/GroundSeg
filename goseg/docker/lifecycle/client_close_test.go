package lifecycle

import "testing"

func TestCloseRuntimeDockerClientAllowsNilClient(t *testing.T) {
	var opErr error
	closeRuntimeDockerClient(nil, "test-op", &opErr)
	if opErr != nil {
		t.Fatalf("expected nil error for nil client close, got %v", opErr)
	}
}
