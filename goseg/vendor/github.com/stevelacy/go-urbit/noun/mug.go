package noun

import (
	"math/big"

	"github.com/twmb/murmur3"
)

var magicSeed1 uint32 = 0xcafebabe // https://github.com/urbit/urbit/blob/1e70a1be2a03a1ba60b7d06ce50f12305be62eb0/pkg/urbit/noun/retrieve.c#L1603
var magicSeed2 uint32 = 0x7fff     // https://github.com/urbit/urbit/blob/1e70a1be2a03a1ba60b7d06ce50f12305be62eb0/pkg/urbit/noun/retrieve.c#L1621
var magicSeed3 uint32 = 0xdeadbeef // https://github.com/urbit/urbit/blob/1e70a1be2a03a1ba60b7d06ce50f12305be62eb0/pkg/urbit/noun/retrieve.c#L1566
var magicSeed4 uint32 = 0xfffe     // https://github.com/urbit/urbit/blob/1e70a1be2a03a1ba60b7d06ce50f12305be62eb0/pkg/urbit/noun/retrieve.c#L1594

func cat(a, b uint32) *big.Int {
	d1 := uint64(b) << 32
	d2 := d1 ^ uint64(a)
	return B(0).SetUint64(d2)
}

func mum(a, b uint32, key *big.Int) uint32 {
	for i := 0; i < 8; i++ {
		m1 := Muk(a, ByteLen(key), key)
		m2 := m1 % (1 << 31)
		m3 := m1 / (1 << 31)

		c1 := m2 ^ m3
		if c1 != 0 {
			return c1
		}
		a++
	}
	return b
}

func Mug(n Noun) uint32 {
	switch t := n.(type) {
	case Atom:
		return mum(magicSeed1, magicSeed2, t.Value)
	case Cell:
		a := Mug(t.Head)
		b := Mug(t.Tail)
		c := cat(a, b)
		return mum(magicSeed3, magicSeed4, c)
	}
	return 1
}

func Muk(seed uint32, length int64, arg *big.Int) uint32 {
	var b2 []byte
	b := BigToLittle(arg)

	if int64(len(b)) < length {
		b2 = make([]byte, length)
		copy(b2, b)
	} else {
		b2 = b
	}

	return murmur3.SeedSum32(seed, b2)
}

// BigToLittle converts from BigEndian to LittleEndian
func BigToLittle(arg *big.Int) []byte {
	b := arg.Bytes()
	// BigEndian to LittleEndian
	for i := 0; i < len(b)/2; i++ {
		j := len(b) - i - 1
		b[i], b[j] = b[j], b[i]
	}
	return b
}

// LittleToBig converts []bytes from BigEndian to LittleEndian
func LittleToBig(b []byte) *big.Int {
	// BigEndian to LittleEndian
	for i := 0; i < len(b)/2; i++ {
		j := len(b) - i - 1
		b[i], b[j] = b[j], b[i]
	}
	b2 := big.NewInt(0)
	b2.SetBytes(b)
	return b2
}
