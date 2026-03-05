package familyspec_test

import (
	"testing"

	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/familyspec"
)

func TestBuildActionContractID(t *testing.T) {
	got := familyspec.BuildActionContractID(
		familycatalog.ProtocolActionContractRoot,
		familycatalog.NamespaceUpload,
		familycatalog.ActionUploadOpenEndpoint,
	)
	want := familycatalog.UploadOpenEndpointContractID
	if got != want {
		t.Fatalf("unexpected action contract id: got %q want %q", got, want)
	}
}
