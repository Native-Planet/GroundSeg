package leak

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/leakchannel"
	"groundseg/structs"
	"time"

	"math/big"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

type leakPayloadType string

const (
	leakPayloadLogin    leakPayloadType = "login"
	leakPayloadLogout   leakPayloadType = "logout"
	leakPayloadPassword leakPayloadType = "password"
	leakPayloadError    leakPayloadType = "protocol_error"
)

func handleAction(patp string, result []byte) {
	statuses := GetLickStatuses()
	isAuth := false
	if status, ok := statuses[patp]; ok {
		isAuth = status.Auth
	}

	if len(result) < 5 {
		zap.L().Warn("Received malformed leak packet: payload too short")
		reportLeakProtocolError(patp, isAuth, "payload too short")
		return
	}
	stripped := result[5:]
	reversed := reverseLittleEndian(stripped)
	jam := new(big.Int).SetBytes(reversed)
	res := noun.Cue(jam)
	cell, ok := res.(noun.Cell)
	if !ok {
		zap.L().Warn(fmt.Sprintf("Received non-cell leak payload from %s: %T", patp, res))
		reportLeakProtocolError(patp, isAuth, fmt.Sprintf("expected packet noun cell, got %T", res))
		return
	}
	bytes, err := decodeAtom(noun.Slag(cell, 1).String())
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to decode payload: %v", err))
		reportLeakProtocolError(patp, isAuth, fmt.Sprintf("failed to decode leak payload: %v", err))
		return
	}
	if err := processAction(patp, bytes); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to process action: %v", err))
	}
}

type leakProtocolErrorPayload struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

func reportLeakProtocolError(patp string, isAuth bool, reason string) {
	payload, err := json.Marshal(leakProtocolErrorPayload{
		Error:  "protocol_error",
		Reason: reason,
	})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to marshal protocol error payload: %v", err))
		return
	}
	sendToLeakChannel(patp, isAuth, leakPayloadError, payload)
}

func parseLeakPayloadType(raw string) (leakPayloadType, bool) {
	kind := leakPayloadType(raw)
	switch kind {
	case leakPayloadLogin, leakPayloadLogout, leakPayloadPassword:
		return kind, true
	default:
		return kind, false
	}
}

func decodeAtom(atom string) ([]byte, error) {
	// Convert string to big.Int
	bigInt := new(big.Int)
	bigInt, ok := bigInt.SetString(atom, 10)
	if !ok {
		return []byte{}, fmt.Errorf("error converting string to big.Int")
	}

	// Convert big.Int to byte array
	byteArray := reverseLittleEndian(bigInt.Bytes())
	return byteArray, nil
}

func reverseLittleEndian(byteSlice []byte) []byte {
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}

func processAction(patp string, byteArray []byte) error {
	var payload structs.GallsegPayload
	if err := json.Unmarshal(byteArray, &payload); err != nil {
		return fmt.Errorf("error unmarshalling payload: %v", err)
	}
	zap.L().Info(fmt.Sprintf("Received gallseg %s action from %s", payload.Payload.Type, patp))
	// handle special cases here, and send everything else to
	// LeakChannel for handler package to process
	status := GetLickStatuses()
	info, exists := status[patp]
	if !exists {
		return nil
	}
	isAuth := info.Auth
	payloadType, knownPayloadType := parseLeakPayloadType(payload.Payload.Type)
	if !knownPayloadType {
		payloadType = leakPayloadType(payload.Payload.Type)
	}
	switch payloadType {
	case leakPayloadLogin:
		if !isAuth {
			urbitLogin(isAuth, patp, byteArray)
		}
	case leakPayloadLogout:
		if isAuth {
			urbitLogout(patp)
		}
	case leakPayloadPassword:
		go func() {
			select {
			case <-leakchannel.Logout:
				urbitLogout(patp)
			case <-time.After(2 * time.Minute):
				zap.L().Error("Password change auto-logout timeout")
			}
		}()
		sendToLeakChannel(patp, isAuth, payloadType, byteArray)
	default:
		sendToLeakChannel(patp, isAuth, payloadType, byteArray)
	}
	return nil
}

func urbitLogout(patp string) {
	lickMu.Lock()
	defer lickMu.Unlock()
	if status, exists := lickStatuses[patp]; exists {
		status.Auth = false
		lickStatuses[patp] = status
	}
}

// login from urbit
func urbitLogin(isAuth bool, patp string, loginPayload []byte) {
	var payload structs.WsLoginPayload
	err := json.Unmarshal(loginPayload, &payload)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Urbit Login failed to unmarshal: %v", err))
		return
	}
	authed := auth.AuthenticateLogin(payload.Payload.Password)

	lickMu.Lock()
	status, exists := lickStatuses[patp]
	if !exists {
		zap.L().Error(fmt.Sprintf("Login failed, %s not in map", patp))
		lickMu.Unlock()
		return
	}
	response := AuthEvent{
		Type:        "urbit-activity",
		PayloadType: string(leakPayloadLogin),
	}
	if authed {
		status.Auth = authed
		lickStatuses[patp] = status
	} else {
		zap.L().Error(fmt.Sprintf("Password incorrect for admin login for  %s", patp))
		response.Error = "Incorrect Password"
	}
	lickMu.Unlock()

	responseBytes, err := json.Marshal(response)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to marshal login response for %s: %v", patp, err))
		return
	}
	channels := GetLickChannels()
	channel, exists := channels[patp]
	if !exists {
		return
	}
	channel <- string(responseBytes)
}

func sendToLeakChannel(patp string, isAuth bool, payloadType leakPayloadType, content []byte) {
	leakchannel.LeakAction <- leakchannel.ActionChannel{
		Patp:    patp,
		Auth:    isAuth,
		Type:    string(payloadType),
		Content: content,
	}
}
