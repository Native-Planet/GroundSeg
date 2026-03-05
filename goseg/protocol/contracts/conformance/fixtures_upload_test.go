package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
)

func TestUploadActionBindingSpecsExposePayloadRules(t *testing.T) {
	specs := contracts.UploadActionBindingSpecs()
	if len(specs) == 0 {
		t.Fatal("expected upload action binding specs")
	}
	seen := make(map[contracts.ActionVerb]struct{}, len(specs))
	for _, spec := range specs {
		if spec.RequiredPayloads.Has(spec.ForbiddenPayloads) {
			t.Fatalf("upload payload rules overlap for %s", spec.Action)
		}
		if spec.RequiredPayloads.IsEmpty() {
			t.Fatalf("upload payload rules missing required payload for %s", spec.Action)
		}
		seen[spec.Action] = struct{}{}
	}
	if _, ok := seen[contracts.ActionUploadOpenEndpoint]; !ok {
		t.Fatalf("missing payload-rule spec for action %s", contracts.ActionUploadOpenEndpoint)
	}
	if _, ok := seen[contracts.ActionUploadReset]; !ok {
		t.Fatalf("missing payload-rule spec for action %s", contracts.ActionUploadReset)
	}
}
