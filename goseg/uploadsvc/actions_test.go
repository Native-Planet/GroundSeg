package uploadsvc

import (
	"testing"
)

func TestParseUploadActionAcceptsKnownActions(t *testing.T) {
	t.Parallel()

	gotOpen, err := ParseUploadAction(string(ActionUploadOpenEndpoint))
	if err != nil {
		t.Fatalf("ParseUploadAction(open-endpoint) returned error: %v", err)
	}
	if gotOpen != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected parsed open-endpoint action: %q", gotOpen)
	}

	gotReset, err := ParseUploadAction(string(ActionUploadReset))
	if err != nil {
		t.Fatalf("ParseUploadAction(reset) returned error: %v", err)
	}
	if gotReset != ActionUploadReset {
		t.Fatalf("unexpected parsed reset action: %q", gotReset)
	}
}

func TestParseUploadActionRejectsUnknownAction(t *testing.T) {
	t.Parallel()

	if _, err := ParseUploadAction("not-an-upload-action"); err == nil {
		t.Fatal("expected unknown upload action to fail parsing")
	}
}
