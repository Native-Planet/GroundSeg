package docker

import "testing"

func TestContains(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	if !contains(items, "beta") {
		t.Fatal("expected contains to find existing element")
	}
	if contains(items, "delta") {
		t.Fatal("expected contains to reject missing element")
	}
}
