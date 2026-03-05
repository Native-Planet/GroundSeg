package governance

import (
	"testing"

	"groundseg/protocol/contracts/catalog/common"
)

func TestStartramContractDeclarationsExposeAPIConnectionContract(t *testing.T) {
	declarations := startramContractDeclarations()
	if len(declarations) != 1 {
		t.Fatalf("expected one startram declaration, got %d", len(declarations))
	}

	declaration := declarations[0]
	if declaration.ID != StartramAPIConnectionErrorID {
		t.Fatalf("unexpected startram contract id: %s", declaration.ID)
	}
	if declaration.Name != StartramAPIConnectionErrorName {
		t.Fatalf("unexpected startram contract name: %s", declaration.Name)
	}
	if declaration.Message != StartramAPIConnectionErrorMessage {
		t.Fatalf("unexpected startram contract message: %s", declaration.Message)
	}
	if declaration.Owner != string(common.OwnerStartram) {
		t.Fatalf("unexpected startram contract owner: %s", declaration.Owner)
	}
}
