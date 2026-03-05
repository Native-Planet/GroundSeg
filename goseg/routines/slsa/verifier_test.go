package slsa

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestCLIArtifactVerifierImplementsInterface(t *testing.T) {
	var _ ArtifactVerifier = CLIArtifactVerifier{}
}

func TestCLIArtifactVerifierVerifyArtifactReturnsErrorForInvalidInputs(t *testing.T) {
	verifier := CLIArtifactVerifier{}
	result, err := verifier.VerifyArtifact(nil, "missing-provenance.intoto.jsonl", "github.com/native-planet/groundseg", []string{"missing-artifact.tar.gz"})
	if err == nil {
		t.Fatal("expected verification to fail for missing provenance/artifact inputs")
	}
	if result != nil {
		value := reflect.ValueOf(result)
		if value.Kind() != reflect.Ptr || !value.IsNil() {
			t.Fatalf("expected nil-equivalent verification result on failure, got %#v", result)
		}
	}
	if !strings.Contains(err.Error(), "missing-provenance") && !strings.Contains(err.Error(), "missing-artifact") {
		t.Fatalf("expected error to include missing input path, got %v", err)
	}
}

func TestCLIArtifactVerifierVerifyArtifactRespectsProvidedContext(t *testing.T) {
	verifier := CLIArtifactVerifier{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := verifier.VerifyArtifact(ctx, "missing-provenance.intoto.jsonl", "github.com/native-planet/groundseg", []string{"missing-artifact.tar.gz"})
	if err == nil {
		t.Fatal("expected canceled context to abort verification")
	}
}
