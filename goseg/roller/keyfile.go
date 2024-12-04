package roller

import (
	"fmt"
	"groundseg/aura"
	"math/big"

	"github.com/stevelacy/go-urbit/noun"
)

// the shape of the keyfile noun
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
