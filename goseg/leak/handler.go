package leak

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/leakchannel"
	"groundseg/structs"
	"reflect"
	"time"

	"math/big"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

func handleAction(patp string, result []byte) {
	stripped := result[5:]
	reversed := reverseLittleEndian(stripped)
	jam := new(big.Int).SetBytes(reversed)
	res := noun.Cue(jam)
	if reflect.TypeOf(res) == reflect.TypeOf(noun.Cell{}) {
		bytes, err := decodeAtom(noun.Slag(res, 1).String())
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to decode payload: %v", err))
			return
		}
		if err := processAction(patp, bytes); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to process action: %v", err))
			return
		}

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
	switch payload.Payload.Type {
	case "login":
		if !isAuth {
			urbitLogin(isAuth, patp, byteArray)
		}
	case "logout":
		if isAuth {
			urbitLogout(patp)
		}
	case "password":
		go func() {
			select {
			case <-leakchannel.Logout:
				urbitLogout(patp)
			case <-time.After(2 * time.Minute):
				zap.L().Error("Password change auto-logout timeout")
			}
		}()
		sendToLeakChannel(patp, isAuth, payload.Payload.Type, byteArray)
	default:
		sendToLeakChannel(patp, isAuth, payload.Payload.Type, byteArray)
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
	defer lickMu.Unlock()
	status, exists := lickStatuses[patp]
	if !exists {
		zap.L().Error(fmt.Sprintf("Login failed, %s not in map", patp))
		return
	}
	response := AuthEvent{
		Type:        "urbit-activity",
		PayloadType: "login",
	}
	if authed {
		status.Auth = authed
		lickStatuses[patp] = status
	} else {
		zap.L().Error(fmt.Sprintf("Password incorrect for admin login for  %s", patp))
		response.Error = "Incorrect Password"
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to marshal login response for %s: %v", patp, err))
		return
	}
	BytesChan[patp] <- string(responseBytes)
}

func sendToLeakChannel(patp string, isAuth bool, payloadType string, content []byte) {
	leakchannel.LeakAction <- leakchannel.ActionChannel{
		Patp:    patp,
		Auth:    isAuth,
		Type:    payloadType,
		Content: content,
	}
}
