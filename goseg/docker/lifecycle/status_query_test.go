package lifecycle

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestNewContainerStatusIndexSupportsExactAndTrimmedNames(t *testing.T) {
	index := NewContainerStatusIndex([]container.Summary{
		{
			Names:  []string{"/vere"},
			Status: "Up 1 minute",
		},
		{
			Names:  []string{"webui"},
			Status: "Exited (0)",
		},
	})

	if status, ok := index["/vere"]; !ok || status != "Up 1 minute" {
		t.Fatalf("expected exact /vere lookup to resolve, got %q (ok=%v)", status, ok)
	}
	if status, ok := index["vere"]; !ok || status != "Up 1 minute" {
		t.Fatalf("expected trimmed vere lookup to resolve, got %q (ok=%v)", status, ok)
	}
	if status, ok := index["webui"]; !ok || status != "Exited (0)" {
		t.Fatalf("expected exact webui lookup to resolve, got %q (ok=%v)", status, ok)
	}
}

func TestResolveStatusesUsesIndexFallbackToCanonicalName(t *testing.T) {
	index := ContainerStatusIndex{
		"/vere": "Up 1 minute",
	}
	resolved := ResolveStatuses(index, []string{"vere", "missing"})
	if resolved["vere"] != "Up 1 minute" {
		t.Fatalf("expected resolved status from canonical key, got %q", resolved["vere"])
	}
	if resolved["missing"] != "not found" {
		t.Fatalf("expected missing status to be not found, got %q", resolved["missing"])
	}
}
