package auth

import (
	"testing"

	"groundseg/auth/tokens"
	"groundseg/structs"
)

func TestUploadAuthPolicyEnforcesTokenValueValidation(t *testing.T) {
	policy := UploadAuthPolicy()
	if !policy.ValidateTokenValue {
		t.Fatal("expected upload auth policy to require token value validation")
	}
}

func TestValidateUploadSessionTokenRequestDetectsContractMismatch(t *testing.T) {
	result := ValidateUploadSessionTokenRequest(
		structs.WsTokenStruct{ID: "session-id", Token: "session-token"},
		structs.WsTokenStruct{ID: "provided-id", Token: "provided-token"},
		nil,
	)
	if result.Status != tokens.UploadValidationStatusTokenContract {
		t.Fatalf("expected token contract mismatch status, got %s", result.Status)
	}
}

func TestValidateUploadSessionTokenRequestRequiresTokenID(t *testing.T) {
	result := ValidateUploadSessionTokenRequest(
		structs.WsTokenStruct{},
		structs.WsTokenStruct{},
		nil,
	)
	if result.Status != tokens.UploadValidationStatusMissingTokenID {
		t.Fatalf("expected missing token ID status, got %s", result.Status)
	}
}
