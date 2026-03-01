package auth

// package for authenticating websockets
// we use a homespun jwt knock-off because no tls on lan
// tokens contain client metadata for authentication
// authentication adds you to the AuthenticatedClients map
// broadcasts get sent to members of this map

// todo: purge old sessions from both maps

// client send:
// {
// 	"type": "verify",
// 	"id": "jsgeneratedid",
// 	"token<optional>": {
// 	  "id": "servergeneratedid",
// 	  "token": "encryptedtext"
// 	}
// }

// 1. we decrypt the token
// 2. we modify token['authorized'] to true
// 3. remove it from 'unauthorized' in system.json
// 4. hash and add to 'authozired' in system.json
// 5. encrypt that, and send it back to the user

// server respond:
// {
// 	"type": "activity",
// 	"response": "ack/nack",
// 	"error": "null/<some_error>",
// 	"id": "jsgeneratedid",
// 	"token": { (either new token or the token the user sent us)
// 	  "id": "relevant_token_id",
// 	  "token": "encrypted_text"
// 	}
// }

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
	"sync"
	"time"

	fernet "github.com/fernet/fernet-go"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	ClientManager = NewClientManager()
	authInitOnce  sync.Once
)

func Initialize() {
	authInitOnce.Do(func() {
		settings := config.AuthSettingsSnapshot()
		authed := settings.AuthorizedSessions
		for key := range authed {
			zap.L().Debug(fmt.Sprintf("Cached auth session: %v", key))
			ClientManager.AddAuthClient(key, &structs.MuConn{Active: false})
		}
		go func() {
			for {
				time.Sleep(10 * time.Minute)
				ClientManager.CleanupStaleSessions(30 * time.Minute)
			}
		}()
	})
}

func NewClientManager() *structs.ClientManager {
	return &structs.ClientManager{
		AuthClients:   make(map[string][]*structs.MuConn),
		UnauthClients: make(map[string][]*structs.MuConn),
	}
}

func GetClientManager() *structs.ClientManager {
	return ClientManager
}

// check if websocket-token pair is auth'd
func WsIsAuthenticated(conn *websocket.Conn, token string) bool {
	return ClientManager.HasAuthConnection(token, conn)
}

// quick check if websocket is authed at all for unauth broadcast (not for auth on its own)
func WsAuthCheck(conn *websocket.Conn) bool {
	return ClientManager.HasAnyAuthConnection(conn)
}

// deactivate ws session
func WsNilSession(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("Invalid session")
	}
	if ClientManager.DeactivateConnection(conn) {
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

// this takes a bool for auth/unauth
// purge token/conn from opposite map
// persists to config
func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	tokenStr := token["token"]
	tokenId := token["id"]
	hashed := sha512.Sum512([]byte(tokenStr))
	hash := hex.EncodeToString(hashed[:])
	muConn := &structs.MuConn{}
	if conn != nil {
		muConn = &structs.MuConn{Conn: conn, Active: true}
		if authed {
			ClientManager.AddAuthClient(tokenId, muConn)
			zap.L().Info(fmt.Sprintf("%s added to auth", tokenId))
		} else {
			ClientManager.AddUnauthClient(tokenId, muConn)
			zap.L().Info(fmt.Sprintf("%s added to unauth", tokenId))
		}
		now := time.Now().Format("2006-01-02_15:04:05")
		return AddSession(tokenId, hash, now, authed)
	} else {
		return fmt.Errorf("Can't add nil session to authmap")
	}
}

// the same but the other way
func RemoveFromAuthMap(tokenId string, fromAuthorized bool) {
	if fromAuthorized {
		ClientManager.Mu.Lock()
		delete(ClientManager.AuthClients, tokenId)
		ClientManager.Mu.Unlock()
	} else {
		ClientManager.Mu.Lock()
		delete(ClientManager.UnauthClients, tokenId)
		ClientManager.Mu.Unlock()
	}
}

// check the validity of the token
func requestIdentityFromRequest(r *http.Request) (string, string) {
	if r == nil {
		return "", ""
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0], r.Header.Get("User-Agent")
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return ip, r.Header.Get("User-Agent")
}

func ValidateAndAuthorizeRequestToken(tokenID, tokenValue string, r *http.Request) (string, error) {
	if tokenValue == "" {
		return "", fmt.Errorf("missing token value")
	}
	key := config.AuthSettingsSnapshot().KeyFile
	res, err := KeyfileDecrypt(tokenValue, key)
	if err != nil {
		return tokenValue, fmt.Errorf("decrypt token: %w", err)
	}
	ip, userAgent := requestIdentityFromRequest(r)
	if !TokenIdAuthed(ClientManager, tokenID) {
		return tokenValue, fmt.Errorf("token id %s is not authorized", tokenID)
	}
	if ip != res["ip"] || userAgent != res["user_agent"] || res["id"] != tokenID {
		return tokenValue, fmt.Errorf("token request context mismatch")
	}
	if res["authorized"] == "true" {
		return tokenValue, nil
	}
	res["authorized"] = "true"
	encryptedText, err := KeyfileEncrypt(res, key)
	if err != nil {
		return tokenValue, fmt.Errorf("encrypt authorized token: %w", err)
	}
	return encryptedText, nil
}

func CheckToken(token map[string]string, r *http.Request) (string, bool) {
	authorizedToken, err := ValidateAndAuthorizeRequestToken(token["id"], token["token"], r)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Invalid token provided: %v", err))
		return token["token"], false
	}
	return authorizedToken, true
}

// make a token authed
func AuthToken(token string) (string, error) {
	key := config.AuthSettingsSnapshot().KeyFile
	res, err := KeyfileDecrypt(token, key)
	if err != nil {
		return "", err
	}
	res["authorized"] = "true"
	encryptedText, err := KeyfileEncrypt(res, key)
	if err != nil {
		zap.L().Error("Error encrypting token")
		return "", err
	}
	return encryptedText, nil
}

// create a new session token
func CreateToken(conn *websocket.Conn, r *http.Request, authed bool) (map[string]string, error) {
	// extract conn info
	var ip string
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	} else {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	userAgent := r.Header.Get("User-Agent")
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
		zap.L().Error(fmt.Sprintf("failed to encrypt token: %v", err))
		return nil, fmt.Errorf("failed to encrypt token: %v", err)
	}
	token := map[string]string{
		"id": id,
	}
	token["token"] = encryptedText
	// Update sessions in the system's configuration
	AddToAuthMap(conn, token, authed)
	return token, nil
}

// take session details and add to SysConfig
func AddSession(tokenID string, hash string, created string, authorized bool) error {
	session := structs.SessionInfo{
		Hash:    hash,
		Created: created,
	}
	if authorized {
		if err := config.UpdateConfTyped(config.WithAuthorizedSession(tokenID, session)); err != nil {
			return fmt.Errorf("Error adding session: %v", err)
		}
		RemoveFromAuthMap(tokenID, false)
	} else {
		if err := config.UpdateConfTyped(config.WithUnauthorizedSession(tokenID, session)); err != nil {
			return fmt.Errorf("Error adding session: %v", err)
		}
		RemoveFromAuthMap(tokenID, true)
	}
	return nil
}

// encrypt the token contents using stored keyfile val
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

// salted sha512
func hashWithSalt(password, salt string) string {
	toHash := salt + password
	res := sha512.Sum512([]byte(toHash))
	return hex.EncodeToString(res[:])
}

func Hasher(password string) string {
	salt := config.AuthSettingsSnapshot().Salt
	return hashWithSalt(password, salt)
}

// check if pw matches sysconfig
func AuthenticateLogin(password string) bool {
	settings := config.AuthSettingsSnapshot()
	hash := hashWithSalt(password, settings.Salt)
	return hash == settings.PasswordHash
}

func LogTokenCheck(token structs.WsTokenStruct, r *http.Request) bool {
	_, err := ValidateAndAuthorizeRequestToken(token.ID, token.Token, r)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Invalid token provided: %v", err))
		return false
	}
	return true
}
