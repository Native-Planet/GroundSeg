package auth

import (
	"context"
	"fmt"
	"net/http"

	"groundseg/auth/lifecycle"
	"groundseg/auth/tokens"
	"groundseg/structs"

	"go.uber.org/zap"
)

// UploadTokenAuthorizationPolicy binds upload-token checking behavior for lifecycle entrypoints.
type UploadTokenAuthorizationPolicy = tokens.UploadTokenAuthorizationPolicy

// UploadTokenAuthorizationResult reports the outcome of an upload token authorization attempt.
type UploadTokenAuthorizationResult = tokens.UploadTokenAuthorizationResult

// UploadAuthPolicy applies a consistent upload policy across lifecycle entrypoints.
// The policy requires token value validation while not requiring request context,
// allowing control-plane setup and data-plane request handling to share the same rules.
func UploadAuthPolicy() UploadTokenAuthorizationPolicy {
	policy := tokens.UploadAuthPolicy()
	policy.ValidateTokenValue = true
	return policy
}

// ValidateUploadSessionToken evaluates token authorization with an explicit policy.
func ValidateUploadSessionToken(
	sessionToken structs.WsTokenStruct,
	providedToken structs.WsTokenStruct,
	r *http.Request,
	policy UploadTokenAuthorizationPolicy,
) UploadTokenAuthorizationResult {
	return tokens.ValidateUploadSessionToken(sessionToken, providedToken, r, policy)
}

// ValidateUploadSessionTokenRequest evaluates token authorization for upload sessions
// with the canonical upload policy.
func ValidateUploadSessionTokenRequest(
	sessionToken structs.WsTokenStruct,
	providedToken structs.WsTokenStruct,
	r *http.Request,
) UploadTokenAuthorizationResult {
	return ValidateUploadSessionToken(sessionToken, providedToken, r, UploadAuthPolicy())
}

func Initialize() error {
	return InitializeWithContext(context.Background())
}

func InitializeWithContext(ctx context.Context) error {
	if err := Start(ctx); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize auth lifecycle: %v", err))
		return err
	}
	return nil
}

func Start(ctx context.Context) error {
	return lifecycle.Start(ctx)
}

func Stop() {
	lifecycle.Stop()
}
