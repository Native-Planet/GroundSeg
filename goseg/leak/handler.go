package leak

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/leakchannel"
	"groundseg/structs"
	"math/big"
	"time"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

type leakPayloadType string

const (
	leakPayloadLogin         leakPayloadType = "login"
	leakPayloadLogout        leakPayloadType = "logout"
	leakPayloadPassword      leakPayloadType = "password"
	leakPayloadProtocolError leakPayloadType = "protocol_error"
	leakPayloadInternal      leakPayloadType = "internal_error"
)
func handleAction(patp string, result []byte) error {
	sessionAuth := isPatpAuthenticated(patp)
	payloadBytes, err := decodePacketPayload(result)
	if err != nil {
		reason, _ := leakProtocolErrorReason(err)
		reportLeakProtocolError(patp, sessionAuth, reason)
		return err
	}
	if err := processAction(patp, payloadBytes); err != nil {
		if reason, isProtocol := leakProtocolErrorReason(err); isProtocol {
			reportLeakProtocolError(patp, sessionAuth, reason)
			return err
		}
		reason, _ := leakInternalErrorReason(err)
		reportLeakInternalError(patp, sessionAuth, reason)
		return leakInternalError{reason: "failed to process action", cause: err}
	}
	return nil
}

type leakProtocolError struct {
	reason string
	cause  error
}

func (e leakProtocolError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.reason, e.cause)
	}
	return e.reason
}

func (e leakProtocolError) Unwrap() error {
	return e.cause
}

type leakInternalError struct {
	reason string
	cause  error
}

func (e leakInternalError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.reason, e.cause)
	}
	return e.reason
}

func (e leakInternalError) Unwrap() error {
	return e.cause
}

func leakProtocolErrorReason(err error) (string, bool) {
	var protocolErr leakProtocolError
	if errors.As(err, &protocolErr) {
		return protocolErr.reason, true
	}
	return "", false
}

func leakInternalErrorReason(err error) (string, bool) {
	var internalErr leakInternalError
	if errors.As(err, &internalErr) {
		return internalErr.reason, true
	}
	return "", false
}

type leakProtocolErrorPayload struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

func reportLeakProtocolError(patp string, isAuth bool, reason string) {
	reportLeakError(patp, isAuth, string(leakPayloadProtocolError), reason)
}

func reportLeakInternalError(patp string, isAuth bool, reason string) {
	reportLeakError(patp, isAuth, string(leakPayloadInternal), reason)
}

func reportLeakError(patp string, isAuth bool, errorType, reason string) {
	payload, err := json.Marshal(leakProtocolErrorPayload{
		Error:  errorType,
		Reason: reason,
	})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to marshal leak error payload: %v", err))
		return
	}
	if err := sendToLeakChannel(patp, isAuth, leakPayloadType(errorType), payload); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to queue leak error for %s: %v", patp, err))
	}
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

func decodePacketPayload(result []byte) ([]byte, error) {
	if len(result) < 5 {
		zap.L().Warn("Received malformed leak packet: payload too short")
		return nil, leakProtocolError{reason: "payload too short"}
	}
	stripped := result[5:]
	jamValue := new(big.Int).SetBytes(reverseLittleEndian(stripped))
	cue := noun.Cue(jamValue)
	cell, ok := cue.(noun.Cell)
	if !ok {
		zap.L().Warn(fmt.Sprintf("Received non-cell leak payload: %T", cue))
		return nil, leakProtocolError{reason: fmt.Sprintf("expected packet noun cell, got %T", cue)}
	}
	payloadBytes, err := decodeAtom(noun.Slag(cell, 1).String())
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to decode leak payload: %v", err))
		return nil, leakProtocolError{reason: "failed to decode leak payload", cause: err}
	}
	return payloadBytes, nil
}

func decodeAtom(atom string) ([]byte, error) {
	bigInt := new(big.Int)
	bigInt, ok := bigInt.SetString(atom, 10)
	if !ok {
		return []byte{}, fmt.Errorf("error converting string to big.Int")
	}
	return reverseLittleEndian(bigInt.Bytes()), nil
}

func reverseLittleEndian(byteSlice []byte) []byte {
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}

func processAction(patp string, byteArray []byte) error {
	payload, err := parseActionPayload(patp, byteArray)
	if err != nil {
		return err
	}
	return routeAction(patp, payload, byteArray)
}

func parseActionPayload(patp string, byteArray []byte) (structs.GallsegPayload, error) {
	var payload structs.GallsegPayload
	if err := json.Unmarshal(byteArray, &payload); err != nil {
		return structs.GallsegPayload{}, leakProtocolError{reason: "error unmarshalling payload", cause: err}
	}
	zap.L().Info(fmt.Sprintf("Received gallseg %s action from %s", payload.Payload.Type, patp))
	return payload, nil
}

func routeAction(patp string, payload structs.GallsegPayload, byteArray []byte) error {
	status, exists := GetLickStatuses()[patp]
	if !exists {
		return leakProtocolError{reason: fmt.Sprintf("unknown ship %q in leak session state", patp)}
	}
	payloadType, _ := parseLeakPayloadType(payload.Payload.Type)
	if payloadType == "" {
		payloadType = leakPayloadType(payload.Payload.Type)
	}
	switch payloadType {
	case leakPayloadLogin:
		if status.Auth {
			return nil
		}
		return urbitLogin(patp, byteArray)
	case leakPayloadLogout:
		if status.Auth {
			urbitLogout(patp)
		}
		return nil
	case leakPayloadPassword:
		watchPasswordLogoutTimeout(patp)
		return sendToLeakChannelWithAuth(patp, status.Auth, payloadType, byteArray)
	default:
		return sendToLeakChannelWithAuth(patp, status.Auth, payloadType, byteArray)
	}
}

func sendToLeakChannelWithAuth(patp string, isAuth bool, payloadType leakPayloadType, content []byte) error {
	return sendToLeakChannel(patp, isAuth, payloadType, content)
}

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

func sendToLeakChannel(patp string, isAuth bool, payloadType leakPayloadType, content []byte) error {
	select {
	case leakchannel.LeakAction <- leakchannel.ActionChannel{
		Patp:    patp,
		Auth:    isAuth,
		Type:    string(payloadType),
		Content: content,
	}:
		return nil
	case <-time.After(2 * time.Second):
		return leakInternalError{reason: "timed out sending leak action"}
	}
}
