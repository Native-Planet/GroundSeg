package leak

import (
	"errors"
	"fmt"
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
