package auth

import (
	"context"
	"fmt"
	"groundseg/auth/lifecycle"
	"groundseg/auth/tokens"
	"groundseg/authsession"
	"groundseg/session"
	"groundseg/structs"
	"net/http"

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
	session.SetClientManager(cm)
}

func IsTokenIdAuthed(token string) bool {
	return TokenIdAuthed(GetClientManager(), token)
}

func GetMuConn(conn *websocket.Conn, tokenId string) *structs.MuConn {
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
	return GetClientManager().HasAuthConnection(token, conn)
}

// quick check if websocket is authed at all for unauth broadcast (not for auth on its own)
func WsAuthCheck(conn *websocket.Conn) bool {
	return GetClientManager().HasAnyAuthConnection(conn)
}

// deactivate ws session
func WsNilSession(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("Invalid session")
	}
	if GetClientManager().DeactivateConnection(conn) {
		return nil
	}
	return fmt.Errorf("Session not in client manager")
}

// is this tokenid in the auth map?
func TokenIdAuthed(clientManager *structs.ClientManager, token string) bool {
	clientManager.Mu.RLock()
	defer clientManager.Mu.RUnlock()
	_, exists := clientManager.AuthClients[token]
	zap.L().Debug(fmt.Sprintf("%s present in authmap: %v", token, exists))
	return exists
}

func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	return authsession.AddToAuthMap(conn, token, authed)
}

// the same but the other way
func RemoveFromAuthMap(tokenId string, fromAuthorized bool) {
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
	return tokens.AuthorizeUploadToken(tokenID, tokenValue, r, policy)
}

func ValidateUploadSessionToken(
	sessionToken structs.WsTokenStruct,
	providedToken structs.WsTokenStruct,
	r *http.Request,
	policy UploadTokenAuthorizationPolicy,
) UploadTokenAuthorizationResult {
	return tokens.ValidateUploadSessionToken(sessionToken, providedToken, r, policy)
}

func ValidateAndAuthorizeRequestToken(tokenID, tokenValue string, r *http.Request) (string, error) {
	return tokens.ValidateAndAuthorizeRequestToken(tokenID, tokenValue, r)
}

func CheckToken(token map[string]string, r *http.Request) (string, bool) {
	return tokens.CheckToken(token, r)
}

// make a token authed
func AuthToken(token string) (string, error) {
	return tokens.AuthToken(token)
}

// create a new session token
func CreateToken(r *http.Request, authed bool) (map[string]string, error) {
	return tokens.CreateToken(r, authed)
}

// encrypt the token contents using stored keyfile val
func KeyfileEncrypt(contents map[string]string, keyStr string) (string, error) {
	return tokens.KeyfileEncrypt(contents, keyStr)
}

func KeyfileDecrypt(tokenStr string, keyStr string) (map[string]string, error) {
	return tokens.KeyfileDecrypt(tokenStr, keyStr)
}

// salted sha512
func Hasher(password string) string {
	return tokens.Hasher(password)
}

// check if pw matches sysconfig
func AuthenticateLogin(password string) bool {
	return tokens.AuthenticateLogin(password)
}

func LogTokenCheck(token structs.WsTokenStruct, r *http.Request) bool {
	return tokens.LogTokenCheck(token, r)
}
