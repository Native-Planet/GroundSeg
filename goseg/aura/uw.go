package aura

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/stevelacy/go-urbit/noun"
)

var (
	mapping = map[int]byte{
		0: '0', 1: '1', 2: '2', 3: '3', 4: '4', 5: '5', 6: '6', 7: '7',
		8: '8', 9: '9', 10: 'a', 11: 'b', 12: 'c', 13: 'd', 14: 'e', 15: 'f',
		16: 'g', 17: 'h', 18: 'i', 19: 'j', 20: 'k', 21: 'l', 22: 'm', 23: 'n',
		24: 'o', 25: 'p', 26: 'q', 27: 'r', 28: 's', 29: 't', 30: 'u', 31: 'v',
		32: 'w', 33: 'x', 34: 'y', 35: 'z', 36: 'A', 37: 'B', 38: 'C', 39: 'D',
		40: 'E', 41: 'F', 42: 'G', 43: 'H', 44: 'I', 45: 'J', 46: 'K', 47: 'L',
		48: 'M', 49: 'N', 50: 'O', 51: 'P', 52: 'Q', 53: 'R', 54: 'S', 55: 'T',
		56: 'U', 57: 'V', 58: 'W', 59: 'X', 60: 'Y', 61: 'Z', 62: '-', 63: '~',
	}
	reverseMapping map[byte]int
)

func init() {
	reverseMapping = make(map[byte]int, len(mapping))
	for i, c := range mapping {
		reverseMapping[c] = i
	}
}

func Uint642Uw(n uint64) string {
	return encodeBigInt(new(big.Int).SetUint64(n))
}

func Uw2Uint64(s string) (uint64, error) {
	value, err := decodeToBigInt(s)
	if err != nil {
		return 0, err
	}
	return value.Uint64(), nil
}

func Noun2Uw(n noun.Noun) (string, error) {
	atom, err := noun.AssertAtom(n)
	if err != nil {
		return "", err
	}
	return encodeBigInt(atom.Value), nil
}

func Uw2Noun(s string) (noun.Noun, error) {
	value, err := decodeToBigInt(s)
	if err != nil {
		return nil, err
	}
	return noun.Atom{Value: value}, nil
}

func Atom2Uw(a noun.Atom) string {
	return encodeBigInt(a.Value)
}

func Jam2Uw(n noun.Noun) string {
	return encodeBigInt(noun.Jam(n))
}

func Uw2Jam(s string) (noun.Noun, error) {
	value, err := decodeToBigInt(s)
	if err != nil {
		return nil, err
	}
	return noun.Cue(value), nil
}

func Cord2Uw(cord string) string {
	if cord == "" {
		return "0w0"
	}
	return encodeBigInt(noun.StringToCord(cord).Value)
}

func Uw2Cord(s string) (string, error) {
	value, err := decodeToBigInt(s)
	if err != nil {
		return "", err
	}
	bytes := value.Bytes()
	for i := 0; i < len(bytes)/2; i++ {
		bytes[i], bytes[len(bytes)-1-i] = bytes[len(bytes)-1-i], bytes[i]
	}
	return string(bytes), nil
}

// helpers

func decodeToBigInt(uw string) (*big.Int, error) {
	if uw == "0w0" { // whats dis
		return big.NewInt(0), nil
	}
	if !strings.HasPrefix(uw, "0w") {
		return nil, fmt.Errorf("invalid @uw encoding: must start with '0w'")
	}
	s := strings.TrimPrefix(uw, "0w")
	s = strings.ReplaceAll(s, ".", "")
	value := new(big.Int)
	for _, c := range s {
		value.Lsh(value, 6)
		v, ok := reverseMapping[byte(c)]
		if !ok {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
		value.Or(value, big.NewInt(int64(v)))
	}
	return value, nil
}

func encodeBigInt(value *big.Int) string {
	if value.Sign() == 0 {
		return "0w0"
	}
	var chars []byte
	tmp := new(big.Int).Set(value)
	for tmp.Sign() > 0 {
		chunk := new(big.Int).And(tmp, big.NewInt(63))
		chars = append(chars, mapping[int(chunk.Int64())])
		tmp.Rsh(tmp, 6)
	}
	for i := 0; i < len(chars)/2; i++ {
		chars[i], chars[len(chars)-1-i] = chars[len(chars)-1-i], chars[i]
	}
	var result []byte
	for i := len(chars) - 1; i >= 0; i-- {
		if (len(chars)-1-i) > 0 && (len(chars)-1-i)%5 == 0 {
			result = append([]byte{'.'}, result...)
		}
		result = append([]byte{chars[i]}, result...)
	}
	return "0w" + string(result)
}
