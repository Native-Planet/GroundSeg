package auth

// package for authenticating websockets
// we use a homespun jwt knock-off because no tls on lan
// tokens contain client metadata for authentication
// authentication adds you to the AuthenticatedClients map
// broadcasts get sent to members of this map

// todo: purge old sessions from both maps

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/structs"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	fernet "github.com/fernet/fernet-go"
	"github.com/gorilla/websocket"
)

var (
	// maps a websocket conn to a tokenid
	// tokenid's can be referenced from the global conf
	AuthenticatedClients = struct {
		Conns map[string]*websocket.Conn
		sync.RWMutex
	}{
		Conns: make(map[string]*websocket.Conn),
	}
	UnauthClients = struct {
		Conns map[string]*websocket.Conn
		sync.RWMutex
	}{
		Conns: make(map[string]*websocket.Conn),
	}
)

// check if websocket-token pair is auth'd
func WsIsAuthenticated(conn *websocket.Conn, token string) bool {
	AuthenticatedClients.RLock()
	defer AuthenticatedClients.RUnlock()
	if AuthenticatedClients.Conns[token] == conn {
		return true
	} else {
		return false
	}
}

// quick check if websocket is authed at all for unauth broadcast (not for auth on its own)
func WsAuthCheck(conn *websocket.Conn) bool {
	AuthenticatedClients.RLock()
	defer AuthenticatedClients.RUnlock()
	for tokenID, con := range AuthenticatedClients.Conns {
		if con == conn {
			config.Logger.Info(fmt.Sprintf("%s is in auth map", tokenID))
			return true
		}
	}
	config.Logger.Info("Client not in auth map")
	return false
}

// is this tokenid in the auth map?
func TokenIdAuthed(token string) bool {
	AuthenticatedClients.RLock()
	defer AuthenticatedClients.RUnlock()
	for tokenID, _ := range AuthenticatedClients.Conns {
		if token == tokenID {
			config.Logger.Info(fmt.Sprintf("%s is in auth map", token))
			return true
		}
	}
	config.Logger.Info("Token not in auth map")
	return false
}

// this takes a bool for auth/unauth -- also persists to config
func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	tokenStr := token["token"]
	tokenId := token["id"]
	hashed := sha512.Sum512([]byte(tokenStr))
	hash := hex.EncodeToString(hashed[:])
	if authed {
		AuthenticatedClients.Lock()
		AuthenticatedClients.Conns[tokenId] = conn
		AuthenticatedClients.Unlock()
		UnauthClients.Lock()
		if _, ok := UnauthClients.Conns[tokenId]; ok {
			delete(UnauthClients.Conns, tokenId)
		}
		UnauthClients.Unlock()
	} else {
		UnauthClients.Lock()
		UnauthClients.Conns[tokenId] = conn
		UnauthClients.Unlock()
		AuthenticatedClients.Lock()
		if _, ok := AuthenticatedClients.Conns[tokenId]; ok {
			delete(AuthenticatedClients.Conns, tokenId)
		}
		AuthenticatedClients.Unlock()
	}
	now := time.Now().Format("2006-01-02_15:04:05")
	err := AddSession(tokenId, hash, now, authed)
	if err != nil {
		return err
	}
	return nil
}

// the same but the other way
func RemoveFromAuthMap(tokenId string, fromAuthorized bool) error {
	if fromAuthorized {
		AuthenticatedClients.Lock()
		if _, ok := AuthenticatedClients.Conns[tokenId]; ok {
			delete(AuthenticatedClients.Conns, tokenId)
		}
		AuthenticatedClients.Unlock()
	} else {
		UnauthClients.Lock()
		if _, ok := UnauthClients.Conns[tokenId]; ok {
			delete(UnauthClients.Conns, tokenId)
		}
		UnauthClients.Unlock()
	}
	return nil
}

// check the validity of the token
func CheckToken(token map[string]string, conn *websocket.Conn, r *http.Request, setup bool) (string, bool) {
	// great you have token. we see if valid.
	if token["token"] == "" {
		return "", false
	}
	config.Logger.Info(fmt.Sprintf("Checking tokenId %s", token["id"]))
	conf := config.Conf()
	key := conf.KeyFile
	res, err := KeyfileDecrypt(token["token"], key)
	if err != nil {
		config.Logger.Warn("Invalid token provided")
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
		if TokenIdAuthed(token["id"]) {
			// check the decrypted token contents
			if ip == res["ip"] && userAgent == res["user_agent"] && res["id"] == token["id"] {
				// already marked authorized? yes
				if res["authorized"] == "true" {
					config.Logger.Info("Token authenticated")
					return token["token"], true
				} else {
					res["authorized"] = "true"
					conf := config.Conf()
					key := conf.KeyFile
					encryptedText, err := KeyfileEncrypt(res, key)
					if err != nil {
						config.Logger.Error("Error encrypting token")
						return token["token"], false
					}
					return encryptedText, true
				}
			} else {
				config.Logger.Warn("TokenId doesn't match session!")
				return token["token"], false
			}
		} else {
			config.Logger.Warn("TokenId isn't an authenticated session")
			return token["token"], false
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
		config.Logger.Error(fmt.Sprintf("failed to encrypt token: %v", err))
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
			"Sessions": map[string]interface{}{
				"Authorized": map[string]structs.SessionInfo{
					tokenID: session,
				},
			},
		}
		if err := config.UpdateConf(update); err != nil {
			return fmt.Errorf("Error adding session: %v", err)
		}
		if err := RemoveFromAuthMap(tokenID, false); err != nil {
			return fmt.Errorf("Error removing session: %v", err)
		}
	} else {
		update := map[string]interface{}{
			"Sessions": map[string]interface{}{
				"Unauthorized": map[string]structs.SessionInfo{
					tokenID: session,
				},
			},
		}
		if err := config.UpdateConf(update); err != nil {
			return fmt.Errorf("Error adding session: %v", err)
		}
		if err := RemoveFromAuthMap(tokenID, true); err != nil {
			return fmt.Errorf("Error removing session: %v", err)
		}
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
	decrypted := fernet.VerifyAndDecrypt([]byte(tokenStr), 60*time.Second, []*fernet.Key{key})
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
