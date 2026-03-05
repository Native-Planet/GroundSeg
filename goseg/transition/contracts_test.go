package transition

import "testing"

func TestParseContainerType(t *testing.T) {
	parsed, ok := ParseContainerType("  VERE ")
	if !ok {
		t.Fatal("expected ParseContainerType to parse normalized values")
	}
	if parsed != ContainerTypeVere {
		t.Fatalf("expected %s, got %s", ContainerTypeVere, parsed)
	}

	if _, ok := ParseContainerType("unknown"); ok {
		t.Fatal("expected ParseContainerType to reject unknown values")
	}
}
