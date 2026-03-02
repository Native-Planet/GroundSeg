package auth

import (
	"context"
	"errors"
	"fmt"
	"groundseg/auth/lifecycle"
	"groundseg/auth/tokens"
	"groundseg/authsession"
	"groundseg/session"
	"groundseg/structs"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// package for authenticating websocket and token flows
var ()

func Initialize() {
	InitializeWithContext(context.Background())
}

func InitializeWithContext(ctx context.Context) {
	if err := Start(ctx); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize auth lifecycle: %v", err))
	}
}

func Start(ctx context.Context) error {
	return lifecycle.Start(ctx)
}

func Stop() {
	lifecycle.Stop()
}

func NewClientManager() *structs.ClientManager {
	return session.NewClientManager()
}

func GetClientManager() *structs.ClientManager {
	return session.GetClientManager()
}

func SetClientManager(cm *structs.ClientManager) {
	if cm == nil {
		zap.L().Warn("ignoring nil auth client manager assignment")
		return
	}
	session.SetClientManager(cm)
}

func IsTokenIdAuthed(token string) bool {
	if strings.TrimSpace(token) == "" {
		return false
	}
	return TokenIdAuthed(GetClientManager(), token)
}

func GetMuConn(conn *websocket.Conn, tokenId string) *structs.MuConn {
	if conn == nil || tokenId == "" {
		return nil
	}
	return GetClientManager().GetMuConn(conn, tokenId)
}

func ReadMuConn(muConn *structs.MuConn) (int, []byte, error) {
	if muConn == nil {
		return 0, nil, fmt.Errorf("invalid websocket session")
	}
	return muConn.Read(GetClientManager())
}

// check if websocket-token pair is auth'd
func WsIsAuthenticated(conn *websocket.Conn, token string) bool {
	if conn == nil || strings.TrimSpace(token) == "" {
		return false
	}
	return GetClientManager().HasAuthConnection(token, conn)
}

// quick check if websocket is authed at all for unauth broadcast (not for auth on its own)
func WsAuthCheck(conn *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	return GetClientManager().HasAnyAuthConnection(conn)
}

// deactivate ws session
func WsNilSession(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("invalid session")
	}
	if GetClientManager().DeactivateConnection(conn) {
		return nil
	}
	return fmt.Errorf("Session not in client manager")
}

// is this tokenid in the auth map?
func TokenIdAuthed(clientManager *structs.ClientManager, token string) bool {
	if clientManager == nil || strings.TrimSpace(token) == "" {
		return false
	}
	exists := clientManager.HasAuthToken(token)
	zap.L().Debug(fmt.Sprintf("%s present in authmap: %v", token, exists))
	return exists
}

func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	if conn == nil {
		return errors.New("missing websocket connection")
	}
	if token == nil {
		return errors.New("missing token map")
	}
	if token["id"] == "" || token["token"] == "" {
		return errors.New("token map must include id and token")
	}
	return authsession.AddToAuthMap(conn, token, authed)
}

// the same but the other way
func RemoveFromAuthMap(tokenId string, fromAuthorized bool) {
	if strings.TrimSpace(tokenId) == "" {
		return
	}
	authsession.RemoveFromAuthMap(tokenId, fromAuthorized)
}

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

// make a token authed
func AuthToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("token is required")
	}
	return tokens.AuthToken(token)
}

// create a new session token
func CreateToken(r *http.Request, authed bool) (map[string]string, error) {
	if r == nil {
		return nil, fmt.Errorf("request context is required")
	}
	return tokens.CreateToken(r, authed)
}

// encrypt the token contents using stored keyfile val
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

// salted sha512
func Hasher(password string) string {
	if password == "" {
		return ""
	}
	return tokens.Hasher(password)
}

// check if pw matches sysconfig
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
