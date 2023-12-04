package leak

import (
	"fmt"
	"goseg/logger"
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
		mark, err := decodeAtom(noun.Snag(res, 0).String())
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to decode mark: %v", err))
		}
		payload, err := decodeAtom(noun.Slag(res, 1).String())
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to payload mark: %v", err))
		}
		logger.Logger.Warn(fmt.Sprintf("mark: %v, payload: %v", mark, payload))
	}
}

func decodeAtom(atom string) (string, error) {
	// Convert string to big.Int
	bigInt := new(big.Int)
	bigInt, ok := bigInt.SetString(atom, 10)
	if !ok {
		return "", fmt.Errorf("error converting string to big.Int")
	}

	// Convert big.Int to byte array
	byteArray := reverseLittleEndian(bigInt.Bytes())

	// Convert bytes to ASCII characters and concatenate
	var asciiStr string
	for _, b := range byteArray {
		asciiStr += string(b)
	}

	return asciiStr, nil
}

func reverseLittleEndian(byteSlice []byte) []byte {
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}
