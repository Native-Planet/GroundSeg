package docker

import (
	"archive/tar"
	"io"
	"testing"

	"groundseg/config"
)

func TestWireguardHostConfigPathSkipsUnknownDockerRoot(t *testing.T) {
	oldDockerDir := config.DockerDir
	t.Cleanup(func() {
		config.DockerDir = oldDockerDir
	})

	config.DockerDir = "/"

	if path := wireguardHostConfigPath(); path != "" {
		t.Fatalf("expected empty host path for unknown docker root, got %q", path)
	}
}

func TestWgConfTarStreamIncludesExpectedFile(t *testing.T) {
	stream, err := wgConfTarStream("wg0.conf", "test-content")
	if err != nil {
		t.Fatalf("failed to create tar stream: %v", err)
	}

	tr := tar.NewReader(stream)
	header, err := tr.Next()
	if err != nil {
		t.Fatalf("failed to read tar header: %v", err)
	}
	if header.Name != "wg0.conf" {
		t.Fatalf("expected wg0.conf entry, got %q", header.Name)
	}

	body, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("failed to read tar body: %v", err)
	}
	if string(body) != "test-content" {
		t.Fatalf("unexpected tar content: %q", string(body))
	}
}
