package leak

import (
	"encoding/json"
	"fmt"
	"groundseg/leakchannel"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

type leakProtocolErrorPayload struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
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
		zap.L().Error(fmt.Sprintf("failed to queue leak error [%s] for %s (authed=%t): %v", errorType, patp, isAuth, err))
	}
}

func sendToLeakChannelWithAuth(patp string, isAuth bool, payloadType leakPayloadType, content []byte) error {
	return sendToLeakChannel(patp, isAuth, payloadType, content)
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
