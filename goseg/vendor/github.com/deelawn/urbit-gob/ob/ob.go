package ob

import (
	"fmt"
	"math/big"

	ugi "github.com/deelawn/urbit-gob/internal"
)

var (
	ux10000               = big.NewInt(0x10000)
	uxFFFFFFFF            = big.NewInt(0xffffffff)
	ux100000000           = big.NewInt(0x100000000)
	uxFFFFFFFFFFFFFFFF, _ = big.NewInt(0).SetString("ffffffffffffffff", 16)
	uxFFFFFFFF00000000, _ = big.NewInt(0).SetString("ffffffff00000000", 16)
	u65535                = big.NewInt(65535)
	u65536                = big.NewInt(65536)
	raku                  = []uint32{0xb76d5eed, 0xee281300, 0x85bcae01, 0x4b387af7}
)

func F(j int, arg *big.Int) *big.Int {

	return muk(raku[j], arg)
}

// TODO: this looping code can be combined into a single loop function that accepts an additional
// function argument. In Fein's case it would be Feis and in Fynd's case it would be Tail.

func Fein(arg string) (*big.Int, error) {

	v, ok := big.NewInt(0).SetString(arg, 10)
	if !ok {
		return nil, fmt.Errorf(ugi.ErrInvalidInt, arg)
	}

	return feinLoop(v)
}

func feinLoop(pyn *big.Int) (*big.Int, error) {

	hi, lo := loopHiLoInit(pyn)

	if pyn.Cmp(ux10000) >= 0 && pyn.Cmp(uxFFFFFFFF) <= 0 {
		v, err := Feis(big.NewInt(0).Sub(pyn, ux10000).String())
		if err != nil {
			return nil, err
		}
		return big.NewInt(0).Add(ux10000, v), nil
	}

	if pyn.Cmp(ux100000000) >= 0 && pyn.Cmp(uxFFFFFFFFFFFFFFFF) <= 0 {
		v, err := feinLoop(lo)
		if err != nil {
			return nil, err
		}
		return big.NewInt(0).Or(hi, v), nil
	}

	return pyn, nil
}

func Fynd(arg *big.Int) (*big.Int, error) {

	return fyndLoop(arg)
}

func fyndLoop(cry *big.Int) (*big.Int, error) {

	hi, lo := loopHiLoInit(cry)

	if cry.Cmp(ux10000) >= 0 && cry.Cmp(uxFFFFFFFF) <= 0 {
		v, err := Tail(big.NewInt(0).Sub(cry, ux10000).String())
		if err != nil {
			return nil, err
		}
		return big.NewInt(0).Add(ux10000, v), nil
	}

	if cry.Cmp(ux100000000) >= 0 && cry.Cmp(uxFFFFFFFFFFFFFFFF) <= 0 {
		v, err := fyndLoop(lo)
		if err != nil {
			return nil, err
		}
		return big.NewInt(0).Or(hi, v), nil
	}

	return cry, nil
}

func loopHiLoInit(v *big.Int) (*big.Int, *big.Int) {

	lo := big.NewInt(0).And(v, uxFFFFFFFF)
	hi := big.NewInt(0).And(v, uxFFFFFFFF00000000)

	return hi, lo
}

func Feis(arg string) (*big.Int, error) {

	v, ok := big.NewInt(0).SetString(arg, 10)
	if !ok {
		return nil, fmt.Errorf(ugi.ErrInvalidInt, arg)
	}

	return Fe(4, u65535, u65536, uxFFFFFFFF, v), nil
}

// TODO: merge Fe and Fen code to accept an additional function argument.

func Fe(
	r int,
	a,
	b,
	k,
	m *big.Int,
) *big.Int {

	c := fe(r, a, b, m)

	if c.Cmp(k) == -1 {
		return c
	}

	return fe(r, a, b, c)
}

func fe(
	r int,
	a,
	b,
	m *big.Int,
) *big.Int {

	left := big.NewInt(0).Mod(m, a)
	right := big.NewInt(0).Div(m, a)

	return feLoop(r, a, b, 1, left, right)
}

func feLoop(
	r int,
	a,
	b *big.Int,
	j int,
	ell,
	arr *big.Int,
) *big.Int {

	if j > r {

		if r%2 != 0 || arr.Cmp(a) == 0 {
			return big.NewInt(0).Add(big.NewInt(0).Mul(a, arr), ell)
		}

		return big.NewInt(0).Add(big.NewInt(0).Mul(a, ell), arr)
	}

	eff := F(j-1, arr)
	tmp := big.NewInt(0).Add(ell, eff)
	if j%2 != 0 {
		tmp = tmp.Mod(tmp, a)
	} else {
		tmp = tmp.Mod(tmp, b)
	}

	return feLoop(r, a, b, j+1, arr, tmp)
}

func Tail(arg string) (*big.Int, error) {

	v, ok := big.NewInt(0).SetString(arg, 10)
	if !ok {
		return nil, fmt.Errorf(ugi.ErrInvalidInt, arg)
	}

	return Fen(4, u65535, u65536, uxFFFFFFFF, v), nil
}

func Fen(
	r int,
	a,
	b,
	k,
	m *big.Int,
) *big.Int {

	c := fen(r, a, b, m)

	if c.Cmp(k) == -1 {
		return c
	}

	return fen(r, a, b, c)
}

func fen(
	r int,
	a,
	b,
	m *big.Int,
) *big.Int {

	ahh := big.NewInt(0).Mod(m, a)
	ale := big.NewInt(0).Div(m, a)
	if r%2 != 0 {
		ahh, ale = ale, ahh
	}

	left := ale
	right := ahh
	if ale.Cmp(a) == 0 {
		left, right = right, left
	}

	return fenLoop(a, b, r, left, right)
}

func fenLoop(
	a,
	b *big.Int,
	j int,
	ell,
	arr *big.Int,
) *big.Int {

	if j < 1 {
		return big.NewInt(0).Add(big.NewInt(0).Mul(a, arr), ell)
	}

	eff := F(j-1, ell)
	tmp := big.NewInt(0)
	useValue := a

	if j%2 == 0 {
		useValue = b
	}

	tmp = tmp.Add(arr, useValue)
	tmp = tmp.Sub(tmp, big.NewInt(0).Mod(eff, useValue))
	tmp = tmp.Mod(tmp, useValue)

	return fenLoop(a, b, j-1, tmp, ell)
}
