package ob

import (
	"math/big"
)

var (
	uxFF   = big.NewInt(0xff)
	uxFF00 = big.NewInt(0xff00)
	u256   = big.NewInt(256)
)

func muk(seed uint32, key *big.Int) *big.Int {

	lo := big.NewInt(0).And(key, uxFF).Int64()
	hi := big.NewInt(0).Div(big.NewInt(0).And(key, uxFF00), u256).Int64()
	hashKey := []rune{rune(lo), rune(hi)}

	hash := murmurHash(hashKey, seed)
	return big.NewInt(int64(hash))
}

func murmurHash(key []rune, seed uint32) uint32 {

	keyLen := len(key)
	remainder := keyLen & 3 // len(key) % 4
	bytes := keyLen - remainder
	h1 := uint32(seed)
	c1 := uint32(0xcc9e2d51)
	c2 := uint32(0x1b873593)
	var (
		i   uint32
		k1  uint32
		h1b uint32
	)

	for int(i) < bytes {

		k1 = (uint32(key[i]) & 0xff) |
			((uint32(key[i+1]) & 0xff) << 8) |
			((uint32(key[i+2]) & 0xff) << 16) |
			((uint32(key[i+3]) & 0xff) << 24)

		i += 4

		k1 = (((k1 & 0xffff) * c1) + ((((k1 >> 16) * c1) & 0xffff) << 16)) & 0xffffffff
		k1 = (k1 << 15) | (k1 >> 17)
		k1 = (((k1 & 0xffff) * c2) + ((((k1 >> 16) * c2) & 0xffff) << 16)) & 0xffffffff

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19)
		h1b = (((h1 & 0xffff) * 5) + ((((h1 >> 16) * 5) & 0xffff) << 16)) & 0xffffffff
		h1 = (((h1b & 0xffff) + 0x6b64) + ((((h1b >> 16) + 0xe654) & 0xffff) << 16))
	}

	k1 = 0

	switch remainder {

	case 3:
		k1 ^= uint32(key[i+2]&0xff) << 16
		fallthrough

	case 2:
		k1 ^= uint32(key[i+1]&0xff) << 8
		fallthrough

	case 1:
		k1 ^= uint32(key[i] & 0xff)
	}

	k1 = (((k1 & 0xffff) * c1) + ((((k1 >> 16) * c1) & 0xffff) << 16)) & 0xffffffff
	k1 = (k1 << 15) | (k1 >> 17)
	k1 = (((k1 & 0xffff) * c2) + ((((k1 >> 16) * c2) & 0xffff) << 16)) & 0xffffffff
	h1 ^= k1

	h1 ^= uint32(keyLen)

	h1 ^= h1 >> 16
	h1 = (((h1 & 0xffff) * 0x85ebca6b) + ((((h1 >> 16) * 0x85ebca6b) & 0xffff) << 16)) & 0xffffffff
	h1 ^= h1 >> 13
	h1 = (((h1 & 0xffff) * 0xc2b2ae35) + ((((h1 >> 16) * 0xc2b2ae35) & 0xffff) << 16)) & 0xffffffff
	h1 ^= h1 >> 16

	return h1
}
