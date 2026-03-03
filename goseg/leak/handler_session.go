package leak

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"groundseg/auth"
	"groundseg/leakchannel"
	"groundseg/structs"
	"time"
)

func watchPasswordLogoutTimeout(patp string) {
	go func() {
		select {
		case <-leakchannel.Logout:
			urbitLogout(patp)
		case <-time.After(2 * time.Minute):
			zap.L().Error("Password change auto-logout timeout")
		}
	}()
}

func isPatpAuthenticated(patp string) bool {
	if status, ok := GetLickStatuses()[patp]; ok {
		return status.Auth
	}
	return false
}

func urbitLogout(patp string) {
	lickMu.Lock()
	defer lickMu.Unlock()
	if status, exists := lickStatuses[patp]; exists {
		status.Auth = false
		lickStatuses[patp] = status
	}
}

func urbitLogin(patp string, loginPayload []byte) error {
	var payload structs.WsLoginPayload
	err := json.Unmarshal(loginPayload, &payload)
	if err != nil {
		return leakProtocolError{reason: "Urbit Login failed to unmarshal", cause: err}
	}
	authed := auth.AuthenticateLogin(payload.Payload.Password)

	lickMu.Lock()
	status, exists := lickStatuses[patp]
	if !exists {
		lickMu.Unlock()
		return leakProtocolError{reason: fmt.Sprintf("Login failed, %s not in map", patp)}
	}
	response := AuthEvent{
		Type:        "urbit-activity",
		PayloadType: string(leakPayloadLogin),
	}
	if authed {
		status.Auth = authed
		lickStatuses[patp] = status
	} else {
		response.Error = "Incorrect Password"
	}
	lickMu.Unlock()

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return leakInternalError{reason: fmt.Sprintf("failed to marshal login response for %s", patp), cause: err}
	}
	channels := GetLickChannels()
	channel, exists := channels[patp]
	if !exists {
		return leakInternalError{reason: fmt.Sprintf("no leak channel for %s", patp)}
	}
	channel <- string(responseBytes)
	return nil
}
