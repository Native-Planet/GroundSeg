package leak

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"math/big"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

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
	jamValue := new(big.Int).SetBytes(reverseLittleEndianBytes(stripped))
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
	return reverseLittleEndianBytes(bigInt.Bytes()), nil
}

func parseActionPayload(patp string, byteArray []byte) (structs.GallsegPayload, error) {
	var payload structs.GallsegPayload
	if err := json.Unmarshal(byteArray, &payload); err != nil {
		return structs.GallsegPayload{}, leakProtocolError{reason: "error unmarshalling payload", cause: err}
	}
	zap.L().Info(fmt.Sprintf("Received gallseg %s action from %s", payload.Payload.Type, patp))
	return payload, nil
}

func processAction(patp string, byteArray []byte) error {
	payload, err := parseActionPayload(patp, byteArray)
	if err != nil {
		return err
	}
	return routeAction(patp, payload, byteArray)
}
