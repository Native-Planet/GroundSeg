package auth

import (
	"fmt"
	"groundseg/auth/tokens"
	"groundseg/structs"
	"net/http"
	"strings"
)

type UploadValidationStatus = tokens.UploadValidationStatus

type UploadTokenAuthorizationPolicy = tokens.UploadTokenAuthorizationPolicy

type UploadTokenAuthorizationResult = tokens.UploadTokenAuthorizationResult

const (
	UploadValidationStatusAuthorized        = tokens.UploadValidationStatusAuthorized
	UploadValidationStatusAuthorizedRotated = tokens.UploadValidationStatusAuthorizedRotated
	UploadValidationStatusMissingTokenValue = tokens.UploadValidationStatusMissingTokenValue
	UploadValidationStatusMissingTokenID    = tokens.UploadValidationStatusMissingTokenID
	UploadValidationStatusNotAuthorized     = tokens.UploadValidationStatusNotAuthorized
	UploadValidationStatusTokenIDMismatch   = tokens.UploadValidationStatusTokenIDMismatch
	UploadValidationStatusTokenContract     = tokens.UploadValidationStatusTokenContract
	UploadValidationStatusContextMismatch   = tokens.UploadValidationStatusContextMismatch
	UploadValidationStatusMalformedToken    = tokens.UploadValidationStatusMalformedToken
	UploadValidationStatusRotationFailed    = tokens.UploadValidationStatusRotationFailed
)

func UploadAuthPolicy() UploadTokenAuthorizationPolicy {
	return tokens.UploadAuthPolicy()
}

func AuthorizeUploadToken(
	tokenID string,
	tokenValue string,
	r *http.Request,
	policy UploadTokenAuthorizationPolicy,
) UploadTokenAuthorizationResult {
	if tokenID == "" {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusMissingTokenID,
			AuthorizationErr: fmt.Errorf("missing token id"),
		}
	}
	if policy.ValidateTokenValue && tokenValue == "" {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusMissingTokenValue,
			AuthorizationErr: fmt.Errorf("missing token value"),
		}
	}
	return tokens.AuthorizeUploadToken(tokenID, tokenValue, r, policy)
}

func ValidateUploadSessionToken(
	sessionToken structs.WsTokenStruct,
	providedToken structs.WsTokenStruct,
	r *http.Request,
	policy UploadTokenAuthorizationPolicy,
) UploadTokenAuthorizationResult {
	if r == nil {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusContextMismatch,
			AuthorizationErr: fmt.Errorf("request context is required"),
		}
	}
	return tokens.ValidateUploadSessionToken(sessionToken, providedToken, r, policy)
}

func ValidateAndAuthorizeRequestToken(tokenID, tokenValue string, r *http.Request) (string, error) {
	if tokenID == "" {
		return tokenValue, fmt.Errorf("token id is required")
	}
	if r == nil {
		return tokenValue, fmt.Errorf("request context is required")
	}
	return tokens.ValidateAndAuthorizeRequestToken(tokenID, tokenValue, r)
}

func CheckToken(token map[string]string, r *http.Request) (string, bool) {
	if token == nil || r == nil {
		return "", false
	}
	return tokens.CheckToken(token, r)
}

func AuthToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("token is required")
	}
	return tokens.AuthToken(token)
}

func CreateToken(r *http.Request, authed bool) (map[string]string, error) {
	if r == nil {
		return nil, fmt.Errorf("request context is required")
	}
	return tokens.CreateToken(r, authed)
}

func KeyfileEncrypt(contents map[string]string, keyStr string) (string, error) {
	if contents == nil || keyStr == "" {
		return "", fmt.Errorf("token contents and key are required")
	}
	return tokens.KeyfileEncrypt(contents, keyStr)
}

func KeyfileDecrypt(tokenStr string, keyStr string) (map[string]string, error) {
	if tokenStr == "" || keyStr == "" {
		return nil, fmt.Errorf("token and key are required")
	}
	return tokens.KeyfileDecrypt(tokenStr, keyStr)
}

func Hasher(password string) string {
	if password == "" {
		return ""
	}
	return tokens.Hasher(password)
}

func AuthenticateLogin(password string) bool {
	if password == "" {
		return false
	}
	return tokens.AuthenticateLogin(password)
}

func LogTokenCheck(token structs.WsTokenStruct, r *http.Request) bool {
	if token == (structs.WsTokenStruct{}) || r == nil {
		return false
	}
	return tokens.LogTokenCheck(token, r)
}
