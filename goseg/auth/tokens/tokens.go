package tokens

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	fernet "github.com/fernet/fernet-go"
)

type UploadValidationStatus string

const (
	UploadValidationStatusAuthorized        UploadValidationStatus = "authorized"
	UploadValidationStatusAuthorizedRotated UploadValidationStatus = "authorized_rotated"
	UploadValidationStatusMissingTokenValue UploadValidationStatus = "missing_token_value"
	UploadValidationStatusMissingTokenID    UploadValidationStatus = "missing_token_id"
	UploadValidationStatusNotAuthorized     UploadValidationStatus = "token_not_authorized"
	UploadValidationStatusTokenIDMismatch   UploadValidationStatus = "token_id_mismatch"
	UploadValidationStatusTokenContract     UploadValidationStatus = "session_token_contract_mismatch"
	UploadValidationStatusContextMismatch   UploadValidationStatus = "token_request_context_mismatch"
	UploadValidationStatusMalformedToken    UploadValidationStatus = "malformed_token"
	UploadValidationStatusRotationFailed    UploadValidationStatus = "token_rotation_failed"
)

type UploadTokenAuthorizationPolicy struct {
	ValidateTokenValue    bool
	RequireRequestContext bool
}

var uploadAuthPolicy = UploadTokenAuthorizationPolicy{
	ValidateTokenValue: false,
}

func UploadAuthPolicy() UploadTokenAuthorizationPolicy {
	return uploadAuthPolicy
}

type UploadTokenAuthorizationResult struct {
	Status           UploadValidationStatus
	AuthorizedToken  string
	AuthorizationErr error
}

func (result UploadTokenAuthorizationResult) IsAuthorized() bool {
	return result.Status == UploadValidationStatusAuthorized ||
		result.Status == UploadValidationStatusAuthorizedRotated
}

func (result UploadTokenAuthorizationResult) IsRotated() bool {
	return result.Status == UploadValidationStatusAuthorizedRotated
}

func requestIdentityFromRequest(r *http.Request) (string, string, error) {
	if r == nil {
		return "", "", fmt.Errorf("request is required")
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0], r.Header.Get("User-Agent"), nil
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return ip, r.Header.Get("User-Agent"), nil
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

	if !IsTokenAuthorized(tokenID) {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusNotAuthorized,
			AuthorizationErr: fmt.Errorf("token id %s is not authorized", tokenID),
		}
	}

	if !policy.ValidateTokenValue {
		return UploadTokenAuthorizationResult{
			Status:          UploadValidationStatusAuthorized,
			AuthorizedToken: tokenValue,
		}
	}

	key := config.AuthSettingsSnapshot().KeyFile
	res, err := KeyfileDecrypt(tokenValue, key)
	if err != nil {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusMalformedToken,
			AuthorizedToken:  tokenValue,
			AuthorizationErr: fmt.Errorf("decrypt token: %w", err),
		}
	}

	if res["id"] != tokenID {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusTokenIDMismatch,
			AuthorizedToken:  tokenValue,
			AuthorizationErr: fmt.Errorf("token id %s does not match token payload", tokenID),
		}
	}

	if policy.RequireRequestContext {
		ip, userAgent, err := requestIdentityFromRequest(r)
		if err != nil {
			return UploadTokenAuthorizationResult{
				Status:           UploadValidationStatusContextMismatch,
				AuthorizedToken:  tokenValue,
				AuthorizationErr: err,
			}
		}
		if ip != res["ip"] || userAgent != res["user_agent"] {
			return UploadTokenAuthorizationResult{
				Status:           UploadValidationStatusContextMismatch,
				AuthorizedToken:  tokenValue,
				AuthorizationErr: fmt.Errorf("token request context mismatch"),
			}
		}
	}

	if res["authorized"] == "true" {
		return UploadTokenAuthorizationResult{
			Status:          UploadValidationStatusAuthorized,
			AuthorizedToken: tokenValue,
		}
	}

	res["authorized"] = "true"
	encryptedText, err := KeyfileEncrypt(res, key)
	if err != nil {
		return UploadTokenAuthorizationResult{
			Status:           UploadValidationStatusRotationFailed,
			AuthorizedToken:  tokenValue,
			AuthorizationErr: fmt.Errorf("encrypt authorized token: %w", err),
		}
	}

	return UploadTokenAuthorizationResult{
		Status:           UploadValidationStatusAuthorizedRotated,
		AuthorizedToken:  encryptedText,
		AuthorizationErr: nil,
	}
}

func IsTokenAuthorized(tokenID string) bool {
	return config.AuthSettingsSnapshot().AuthorizedSessions != nil && isTokenInMap(config.AuthSettingsSnapshot().AuthorizedSessions, tokenID)
}

func isTokenInMap(s map[string]structs.SessionInfo, tokenID string) bool {
	_, exists := s[tokenID]
	return exists
}

func ValidateUploadSessionToken(
	sessionToken structs.WsTokenStruct,
	providedToken structs.WsTokenStruct,
	r *http.Request,
	policy UploadTokenAuthorizationPolicy,
) UploadTokenAuthorizationResult {
	if sessionToken != (structs.WsTokenStruct{}) {
		if sessionToken.ID != providedToken.ID || sessionToken.Token != providedToken.Token {
			return UploadTokenAuthorizationResult{
				Status:           UploadValidationStatusTokenContract,
				AuthorizedToken:  providedToken.Token,
				AuthorizationErr: fmt.Errorf("upload token does not match upload session"),
			}
		}
	}

	return AuthorizeUploadToken(providedToken.ID, providedToken.Token, r, policy)
}

func ValidateAndAuthorizeRequestToken(tokenID, tokenValue string, r *http.Request) (string, error) {
	eval := AuthorizeUploadToken(tokenID, tokenValue, r, UploadAuthPolicy())
	if !eval.IsAuthorized() {
		return tokenValue, eval.AuthorizationErr
	}
	return eval.AuthorizedToken, nil
}

func CheckToken(token map[string]string, r *http.Request) (string, bool) {
	authorizedToken, err := ValidateAndAuthorizeRequestToken(token["id"], token["token"], r)
	if err != nil {
		return token["token"], false
	}
	return authorizedToken, true
}

func AuthToken(token string) (string, error) {
	key := config.AuthSettingsSnapshot().KeyFile
	res, err := KeyfileDecrypt(token, key)
	if err != nil {
		return "", err
	}
	res["authorized"] = "true"
	encryptedText, err := KeyfileEncrypt(res, key)
	if err != nil {
		return "", err
	}
	return encryptedText, nil
}

func CreateToken(r *http.Request, authed bool) (map[string]string, error) {
	// extract conn info
	ip, userAgent, err := requestIdentityFromRequest(r)
	if err != nil {
		return nil, err
	}
	settings := config.AuthSettingsSnapshot()
	now := time.Now().Format("2006-01-02_15:04:05")
	// generate random strings for id, secret, and padding
	id := config.RandString(32)
	secret := config.RandString(128)
	padding := config.RandString(32)
	contents := map[string]string{
		"id":         id,
		"ip":         ip,
		"user_agent": userAgent,
		"secret":     secret,
		"padding":    padding,
		"authorized": fmt.Sprintf("%v", authed),
		"created":    now,
	}
	// encrypt the contents
	key := settings.KeyFile
	encryptedText, err := KeyfileEncrypt(contents, key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %v", err)
	}
	token := map[string]string{
		"id": id,
	}
	token["token"] = encryptedText
	return token, nil
}

func KeyfileEncrypt(contents map[string]string, keyStr string) (string, error) {
	fileBytes, err := ioutil.ReadFile(keyStr)
	if err != nil {
		return "", err
	}
	contentBytes, err := json.Marshal(contents)
	if err != nil {
		return "", err
	}
	key, err := fernet.DecodeKey(string(fileBytes))
	if err != nil {
		return "", err
	}
	tok, err := fernet.EncryptAndSign(contentBytes, key)
	if err != nil {
		return "", err
	}
	return string(tok), nil
}

func KeyfileDecrypt(tokenStr string, keyStr string) (map[string]string, error) {
	fileBytes, err := ioutil.ReadFile(keyStr)
	if err != nil {
		return nil, err
	}
	key, err := fernet.DecodeKey(string(fileBytes))
	if err != nil {
		return nil, err
	}
	decrypted := fernet.VerifyAndDecrypt([]byte(tokenStr), 0, []*fernet.Key{key})
	if decrypted == nil {
		return nil, fmt.Errorf("verification or decryption failed")
	}
	var contents map[string]string
	err = json.Unmarshal(decrypted, &contents)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func hashWithSalt(password, salt string) string {
	toHash := salt + password
	res := sha512.Sum512([]byte(toHash))
	return hex.EncodeToString(res[:])
}

func Hasher(password string) string {
	salt := config.AuthSettingsSnapshot().Salt
	return hashWithSalt(password, salt)
}

func AuthenticateLogin(password string) bool {
	settings := config.AuthSettingsSnapshot()
	hash := hashWithSalt(password, settings.Salt)
	return hash == settings.PasswordHash
}

func LogTokenCheck(token structs.WsTokenStruct, r *http.Request) bool {
	_, err := ValidateAndAuthorizeRequestToken(token.ID, token.Token, r)
	return err == nil
}
