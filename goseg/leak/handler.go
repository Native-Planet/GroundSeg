package leak

import (
	"encoding/json"
	"fmt"
	"goseg/leakchannel"
	"goseg/logger"
	"goseg/structs"
	"reflect"

	"math/big"

	"github.com/stevelacy/go-urbit/noun"
)

func handleAction(result []byte) {
	stripped := result[5:]
	reversed := reverseLittleEndian(stripped)
	jam := new(big.Int).SetBytes(reversed)
	res := noun.Cue(jam)
	if reflect.TypeOf(res) == reflect.TypeOf(noun.Cell{}) {
		err := decodeAtom(noun.Slag(res, 1).String())
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to decode payload: %v", err))
			return
		}
	}
}

func decodeAtom(atom string) error {
	// Convert string to big.Int
	bigInt := new(big.Int)
	bigInt, ok := bigInt.SetString(atom, 10)
	if !ok {
		return fmt.Errorf("error converting string to big.Int")
	}

	// Convert big.Int to byte array
	byteArray := reverseLittleEndian(bigInt.Bytes())

	var payload structs.GallsegPayload
	if err := json.Unmarshal(byteArray, &payload); err != nil {
		return fmt.Errorf("error unmarshalling payload: %v", err)
	}
	// handle special cases here, and send everything else to
	// LeakChannel for handler package to process
	switch payload.Payload.Type {
	case "login":
		urbitLogin(byteArray)
	default:
		sendToLeakChannel(payload.Payload.Type, byteArray)
	}
	return nil
}

func sendToLeakChannel(payloadType string, content []byte) {
	leakchannel.LeakAction <- leakchannel.ActionChannel{
		Type:    payloadType,
		Content: content,
	}
}

func reverseLittleEndian(byteSlice []byte) []byte {
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}

// login from urbit
func urbitLogin(loginPayload []byte) {
	var payload structs.WsLoginPayload
	err := json.Unmarshal(loginPayload, &payload)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Urbit Login failed to unmarshal: %v", err))
		return
	}
	logger.Logger.Warn(fmt.Sprintf("Login Payload:::: %+v", payload))
}
