package roller

import (
	"fmt"
	"groundseg/aura"
	"math/big"

	"github.com/stevelacy/go-urbit/noun"
)

// aaaaa
func xor(a, b []byte) []byte {
	length := len(a)
	if len(b) > length {
		length = len(b)
	}
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		var aByte, bByte byte
		if i < len(a) {
			aByte = a[i]
		}
		if i < len(b) {
			bByte = b[i]
		}
		result[i] = aByte ^ bByte
	}
	return result
}

// encode jammed noun to hex string
func bytes2Hex(bytes []byte) string {
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	hexStr := fmt.Sprintf("%x", bytes)
	if len(bytes) == 0 {
		return "0"
	}
	return hexStr
}

func keyNoun(point *big.Int, revision int, bnsec *big.Int) noun.Cell {
	return noun.MakeNoun([]interface{}{
		[]interface{}{1, 0},            // [1 0]
		point,                          // point/ship number
		[]interface{}{revision, bnsec}, // [revision bnsec]
		0,                              // final 0
	}).(noun.Cell)
}

// generate a @uw keyfile
func Keyfile(crypt, auth string, ship interface{}, revision int) (string, error) {
	point, err := ValidatePoint(ship, false)
	if err != nil {
		return "", err
	}
	if ship == nil {
		return "", fmt.Errorf("invalid point (nil pointer)")
	}
	ring := crypt + auth + "42"
	bnsec := new(big.Int)
	bnsec, ok := bnsec.SetString(ring, 16)
	if !ok {
		return "", fmt.Errorf("failed to parse ring as hex: %s", ring)
	}
	sed := keyNoun(point, revision, bnsec)
	jammed := noun.Jam(sed)
	jammedNoun := deref(jammed)
	encoded := aura.Big2Uw(jammedNoun)
	return encoded, nil
}

func deref(n *big.Int) *big.Int {
	result := new(big.Int)
	return result.Set(n)
}
