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

// generate a @uw keyfile
func Keyfile(crypt, auth string, ship interface{}, revision int) (string, error) {
	point, err := ValidatePoint(ship, false)
	if err != nil {
		return "", err
	}
	if ship == nil {
		return "", fmt.Errorf("Invalid point; nil pointer")
	}
	ring := crypt + auth + "42"
	bnsec := new(big.Int)
	bnsec.SetString(ring, 16)
	sed := noun.Cell{
		Head: noun.MakeNoun(*point),
		Tail: noun.Cell{
			Head: noun.MakeNoun(revision),
			Tail: noun.Cell{
				Head: noun.MakeNoun(bnsec),
				Tail: noun.MakeNoun(1),
			},
		},
	}
	jammed := noun.Jam(sed)
	bytes := jammed.Bytes()
	hexStr := bytes2Hex(bytes)
	encoded := aura.Cord2Uw(hexStr)
	return encoded, nil
}
