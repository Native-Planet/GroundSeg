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
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	fernet "github.com/fernet/fernet-go"
	"github.com/gorilla/websocket"
)

func NewClientManager() *structs.ClientManager {
	return &structs.ClientManager{
		AuthClients:   make(map[string]*structs.MuConn),
		UnauthClients: make(map[string]*structs.MuConn),
	}
}

var ClientManager = NewClientManager()

func init() {
	conf := config.Conf()
	authed := conf.Sessions.Authorized
	logger.Logger.Info(fmt.Sprintf("Auth config: %v",authed))
	for key := range authed {
		logger.Logger.Info(fmt.Sprintf("Loading saved token session %v",key))
		ClientManager.AddAuthClient(key, nil)
		AddToAuthMap(nil, key, true)
	}
}

// check if websocket-token pair is auth'd
func WsIsAuthenticated(conn *websocket.Conn, token string) bool {
	ClientManager.Mu.RLock()
	defer ClientManager.Mu.RUnlock()
	if ClientManager.AuthClients[token].Conn == conn {
		return true
	} else {
		return false
	}
}

// quick check if websocket is authed at all for unauth broadcast (not for auth on its own)
func WsAuthCheck(conn *websocket.Conn) bool {
	ClientManager.Mu.RLock()
	defer ClientManager.Mu.RUnlock()
	for _, client := range ClientManager.AuthClients {
		if client.Conn == conn {
			logger.Logger.Info("Client is in auth map")
			return true
		}
	}
	logger.Logger.Info("Client not in auth map")
	return false
}

// deactivate ws session
func WsNilSession(conn *websocket.Conn) error {
	if WsAuthCheck(conn) {
		ClientManager.Mu.Lock()
		defer ClientManager.Mu.Unlock()
		for _, client := range ClientManager.AuthClients {
			if client.Conn == conn {
				client.Active = false
				return nil
			}
		}
	} else {
		ClientManager.Mu.Lock()
		defer ClientManager.Mu.Unlock()
		for _, client := range ClientManager.UnauthClients {
			if client.Conn == conn {
				client.Active = false
				return nil
			}
		}
	}
	return fmt.Errorf("Session not in client manager")
}

// is this tokenid in the auth map?
func TokenIdAuthed(clientManager *structs.ClientManager, token string) bool {
	ClientManager.Mu.RLock()
	defer ClientManager.Mu.RUnlock()
	_, exists := ClientManager.AuthClients[token]
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
	muConn := &structs.MuConn{Conn: conn}
	if authed {
		ClientManager.AddAuthClient(tokenId, muConn)
		logger.Logger.Info(fmt.Sprintf("%s added to auth", tokenId))
	} else {
		ClientManager.AddUnauthClient(tokenId, muConn)
		logger.Logger.Info(fmt.Sprintf("%s added to unauth", tokenId))
	}
	now := time.Now().Format("2006-01-02_15:04:05")
	return AddSession(tokenId, hash, now, authed)
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
func CheckToken(token map[string]string, conn *websocket.Conn, r *http.Request) (string, bool) {
	// great you have token. we see if valid.
	if token["token"] == "" {
		return "", false
	}
	logger.Logger.Info(fmt.Sprintf("Checking tokenId %s", token["id"]))
	conf := config.Conf()
	key := conf.KeyFile
	res, err := KeyfileDecrypt(token["token"], key)
	if err != nil {
		logger.Logger.Warn(fmt.Sprintf("Invalid token provided: %v", err))
		return token["token"], false
	} else {
		// so you decrypt. now we see the useragent and ip.
		var ip string
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		} else {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		userAgent := r.Header.Get("User-Agent")
		// you in auth map?
		if TokenIdAuthed(ClientManager, token["id"]) {
			// check the decrypted token contents
			if ip == res["ip"] && userAgent == res["user_agent"] && res["id"] == token["id"] {
				// already marked authorized? yes
				if res["authorized"] == "true" {
					logger.Logger.Info("Token authenticated")
					return token["token"], true
				} else {
					res["authorized"] = "true"
					encryptedText, err := KeyfileEncrypt(res, key)
					if err != nil {
						logger.Logger.Error("Error encrypting token")
						return token["token"], false
					}
					return encryptedText, true
				}
			} else {
				logger.Logger.Warn("TokenId doesn't match session!")
				logger.Logger.Info(fmt.Sprintf("%v:%v -- %v:%v -- %v:%v -- %v:%v", ip, res["ip"], userAgent, res["user_agent"], res["id"], token["id"]))
				return token["token"], false
			}
		} else {
			logger.Logger.Info(fmt.Sprintf("Token %v failed TokenIdAuthed",token["id"]))
		}
	}
	return token["token"], false
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
	conf := config.Conf()
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
	key := conf.KeyFile
	encryptedText, err := KeyfileEncrypt(contents, key)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("failed to encrypt token: %v", err))
		return nil, fmt.Errorf("failed to encrypt token: %v", err)
	}
	token := map[string]string{
		"id":    id,
		"token": encryptedText,
	}
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
		update := map[string]interface{}{
			"sessions": map[string]interface{}{
				"authorized": map[string]structs.SessionInfo{
					tokenID: session,
				},
			},
		}
		if err := config.UpdateConf(update); err != nil {
			return fmt.Errorf("Error adding session: %v", err)
		}
		RemoveFromAuthMap(tokenID, false)
	} else {
		update := map[string]interface{}{
			"sessions": map[string]interface{}{
				"unauthorized": map[string]structs.SessionInfo{
					tokenID: session,
				},
			},
		}
		if err := config.UpdateConf(update); err != nil {
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
func Hasher(password string) string {
	conf := config.Conf()
	salt := conf.Salt
	toHash := salt + password
	res := sha512.Sum512([]byte(toHash))
	return hex.EncodeToString(res[:])
}

// check if pw matches sysconfig
func AuthenticateLogin(password string) bool {
	conf := config.Conf()
	hash := Hasher(password)
	if hash == conf.PwHash {
		return true
	} else {
		return false
	}
}
