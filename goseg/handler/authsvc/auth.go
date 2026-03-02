package authsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/authsession"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/leakchannel"
	"groundseg/structs"
	"groundseg/system"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	lockoutStateMu sync.RWMutex
	lockoutState   = &lockoutStats{}
)

const (
	MaxFailedLogins = 5
	LockoutDuration = 2 * time.Minute
)

type lockoutStats struct {
	failedLogins int
	remainder    int
}

type lockoutStateSnapshot struct {
	FailedLogins int
	Remainder    int
}

func getLockoutStateSnapshot() lockoutStateSnapshot {
	lockoutStateMu.RLock()
	defer lockoutStateMu.RUnlock()
	return lockoutStateSnapshot{
		FailedLogins: lockoutState.failedLogins,
		Remainder:    lockoutState.remainder,
	}
}

func resetFailedLoginState() {
	lockoutStateMu.Lock()
	defer lockoutStateMu.Unlock()
	lockoutState.failedLogins = 0
	lockoutState.remainder = 0
}

func registerFailedLogin() bool {
	lockoutStateMu.Lock()
	defer lockoutStateMu.Unlock()
	lockoutState.failedLogins++
	return lockoutState.failedLogins >= MaxFailedLogins && lockoutState.remainder == 0
}

func setLockoutRemainder(value int) {
	lockoutStateMu.Lock()
	defer lockoutStateMu.Unlock()
	lockoutState.remainder = value
}

func decrementAndGetRemainder() int {
	lockoutStateMu.Lock()
	defer lockoutStateMu.Unlock()
	if lockoutState.remainder > 0 {
		lockoutState.remainder--
	}
	return lockoutState.remainder
}

var ErrInvalidLoginCredentials = errors.New("invalid login credentials")

func LoginHandler(conn *structs.MuConn, msg []byte) (map[string]string, error) {
	var loginPayload structs.WsLoginPayload
	err := json.Unmarshal(msg, &loginPayload)
	if err != nil {
		return make(map[string]string), fmt.Errorf("Couldn't unmarshal login payload: %w", err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Payload.Password)
	if isAuthenticated {
		resetFailedLoginState()
		newToken, err := auth.AuthToken(loginPayload.Token.Token)
		if err != nil {
			return make(map[string]string), err
		}
		token := map[string]string{
			"id":    loginPayload.Token.ID,
			"token": newToken,
		}
		if err := authsession.AddToAuthMap(conn.Conn, token, true); err != nil {
			return make(map[string]string), fmt.Errorf("Unable to process login: %w", err)
		}
		zap.L().Info(fmt.Sprintf("Session %s logged in", loginPayload.Token.ID))
		return token, nil
	}

	shouldStartLockout := registerFailedLogin()
	zap.L().Warn(fmt.Sprintf("Failed auth"))
	if shouldStartLockout {
		go enforceLockout()
	}
	loginError, _ := json.Marshal(map[string]string{
		"type":    "login-failed",
		"message": "Login failed. Please try again.",
	})
	conn.Write(loginError)
	return map[string]string{}, ErrInvalidLoginCredentials
}

func UnauthHandler() ([]byte, error) {
	state := getLockoutStateSnapshot()
	blob := structs.UnauthBroadcast{
		Type:      "structure",
		AuthLevel: "unauthorized",
		Login: struct {
			Remainder int `json:"remainder"`
		}{
			Remainder: state.Remainder,
		},
	}
	resp, err := json.Marshal(blob)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling message: %w", err)
	}
	return resp, nil
}

func enforceLockout() {
	// todo: extend remainder
	setLockoutRemainder(120)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for getLockoutStateSnapshot().Remainder > 0 {
		unauth, err := UnauthHandler()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
		}
		broadcast.UnauthBroadcast(unauth)
		<-ticker.C
		decrementAndGetRemainder()
	}
	resetFailedLoginState()

	unauth, err := UnauthHandler()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
	}
	broadcast.UnauthBroadcast(unauth)
}

func LogoutHandler(msg []byte) error {
	var logoutPayload structs.WsLogoutPayload
	err := json.Unmarshal(msg, &logoutPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %w", err)
	}
	authsession.RemoveFromAuthMap(logoutPayload.Token.ID, true)
	return nil
}

func C2CHandler(msg []byte) ([]byte, error) {
	var c2cPayload structs.WsC2cPayload
	err := json.Unmarshal(msg, &c2cPayload)
	if err != nil {
		return nil, fmt.Errorf("Couldn't unmarshal c2c payload: %w", err)
	}
	var resp []byte
	switch c2cPayload.Payload.Type {
	case "c2c":
		if err := system.C2CConnect(c2cPayload.Payload.SSID, c2cPayload.Payload.Password); err != nil {
			return nil, fmt.Errorf("c2c connect failed: %w", err)
		}
	case "wifi":
		return nil, errors.New("unsupported c2c action")
	case "wifis":
		return nil, errors.New("unsupported c2c action")
	case "wss":
		return nil, errors.New("unsupported c2c action")
	default:
		return nil, fmt.Errorf("Invalid c2c request")
		/*
			blob := structs.C2CBroadcast{
				Type:  "c2c",
				SSIDS: system.C2CStoredSSIDs,
			}
			resp, err = json.Marshal(blob)
			if err != nil {
				return nil, fmt.Errorf("Error unmarshalling message: %w", err)
			}
		*/
	}
	return resp, nil
}

func PwHandler(msg []byte, urbitMode bool) error {
	var pwPayload structs.WsPwPayload
	err := json.Unmarshal(msg, &pwPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal password payload: %w", err)
	}
	switch pwPayload.Payload.Action {
	case "modify":
		zap.L().Info("Setting new password")
		conf := config.Conf()
		if auth.Hasher(pwPayload.Payload.Old) == conf.PwHash {
			if err := config.UpdateConfTyped(config.WithPwHash(auth.Hasher(pwPayload.Payload.Password))); err != nil {
				return fmt.Errorf("Unable to update password: %w", err)
			}
			if urbitMode {
				leakchannel.Logout <- struct{}{}
				return nil
			}
			if pwPayload.Token.ID == "" {
				return fmt.Errorf("Missing token id for logout after password update")
			}
			authsession.RemoveFromAuthMap(pwPayload.Token.ID, true)
			return nil
		}
		return fmt.Errorf("Current password is incorrect")
	default:
		return fmt.Errorf("Unrecognized password action: %v", pwPayload.Payload.Action)
	}
}
