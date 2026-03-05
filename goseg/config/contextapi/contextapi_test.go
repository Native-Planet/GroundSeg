package contextapi

import "testing"

func TestContextAPIRoundTrip(t *testing.T) {
	original := Snapshot()
	t.Cleanup(func() {
		Set(original)
	})

	SetBasePath("/tmp/groundseg-contextapi")
	SetArchitecture("arm64")
	SetDebugMode(true)
	SetDockerDir("/tmp/groundseg-contextapi/docker/")

	snapshot := Snapshot()
	if snapshot.BasePath != "/tmp/groundseg-contextapi" {
		t.Fatalf("unexpected base path: %q", snapshot.BasePath)
	}
	if snapshot.Architecture != "arm64" {
		t.Fatalf("unexpected architecture: %q", snapshot.Architecture)
	}
	if !snapshot.DebugMode {
		t.Fatal("expected debug mode to be true")
	}
	if snapshot.DockerDir != "/tmp/groundseg-contextapi/docker/" {
		t.Fatalf("unexpected docker dir: %q", snapshot.DockerDir)
	}
}
