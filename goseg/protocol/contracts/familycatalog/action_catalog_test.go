package familycatalog

import (
	"reflect"
	"testing"

	catalogaction "groundseg/protocol/contracts/catalog/action"
)

func TestActionCatalogCompatibilityWrappers(t *testing.T) {
	if !reflect.DeepEqual(ProtocolActionSpecs(), catalogaction.ProtocolActionSpecs()) {
		t.Fatalf("protocol action specs wrapper diverged from catalog/action implementation")
	}
	if !reflect.DeepEqual(UploadActionSpecs(), catalogaction.UploadActionSpecs()) {
		t.Fatalf("upload action specs wrapper diverged from catalog/action implementation")
	}
	if !reflect.DeepEqual(AllActionSpecs(), catalogaction.AllActionSpecs()) {
		t.Fatalf("all action specs wrapper diverged from catalog/action implementation")
	}
}
