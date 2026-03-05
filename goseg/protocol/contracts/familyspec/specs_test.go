package familyspec

import "testing"

func TestBuildActionContractID(t *testing.T) {
	got := BuildActionContractID("protocol.actions", "upload", "open-endpoint")
	want := "protocol.actions.upload.open-endpoint"
	if got != want {
		t.Fatalf("unexpected action contract id: got %q want %q", got, want)
	}
}
