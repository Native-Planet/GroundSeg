package slsa

import (
	"context"

	"github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier/verify"
)

type ArtifactVerifier interface {
	VerifyArtifact(ctx context.Context, provenancePath string, sourceURI string, artifacts []string) (any, error)
}

type CLIArtifactVerifier struct{}

func (CLIArtifactVerifier) VerifyArtifact(ctx context.Context, provenancePath string, sourceURI string, artifacts []string) (any, error) {
	verifyCmd := &verify.VerifyArtifactCommand{
		ProvenancePath:  provenancePath,
		SourceURI:       sourceURI,
		PrintProvenance: false,
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return verifyCmd.Exec(ctx, artifacts)
}
