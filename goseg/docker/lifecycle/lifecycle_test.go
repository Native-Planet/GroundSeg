package lifecycle

import "testing"

func TestImageTagFromReferenceEdgeCases(t *testing.T) {
	if tag := imageTagFromReference("groundseg:0.1.0"); tag != "0.1.0" {
		t.Fatalf("expected tag extracted, got %q", tag)
	}
	if tag := imageTagFromReference("groundseg"); tag != "" {
		t.Fatalf("expected empty tag for no-tag image, got %q", tag)
	}
	if tag := imageTagFromReference("localhost:5000/team/image@sha256:abc"); tag != "" {
		t.Fatalf("expected digest-only image without tag to be empty, got %q", tag)
	}
}

func TestLifecyclePackageTagFromReferenceWithRepoSlashWithoutTag(t *testing.T) {
	if tag := imageTagFromReference("repo.local/team/image@sha256:abc"); tag != "" {
		t.Fatalf("expected empty tag when image contains digest only, got %q", tag)
	}
}
