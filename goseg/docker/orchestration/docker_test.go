package orchestration

import (
	"errors"
	"testing"

	"github.com/docker/docker/client"

	"groundseg/docker/lifecycle"
)

func TestContains(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	if !Contains(items, "beta") {
		t.Fatal("expected contains to find existing element")
	}
	if Contains(items, "delta") {
		t.Fatal("expected contains to reject missing element")
	}
}

func TestGetContainerRunningStatusReturnsInitError(t *testing.T) {
	rt := &orchestrationRuntime{
		lifecycleRuntime: lifecycle.NewRuntime(
			lifecycle.WithDockerClientFactory(func(_ ...client.Opt) (*client.Client, error) {
				return nil, errors.New("cannot init docker client")
			}),
		),
	}

	if _, err := rt.getContainerRunningStatus("test-container"); err == nil {
		t.Fatalf("expected init failure to surface as an error")
	}
}
