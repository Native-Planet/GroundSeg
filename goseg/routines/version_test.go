package routines

import (
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestVersionComponentNameUsesAgentJSONSlot(t *testing.T) {
	field, ok := reflect.TypeFor[structs.Channel]().FieldByName("BeszelAgent")
	if !ok {
		t.Fatal("BeszelAgent version field not found")
	}
	if got := versionComponentName(field); got != "beszel-agent" {
		t.Fatalf("versionComponentName() = %q, want %q", got, "beszel-agent")
	}
}
