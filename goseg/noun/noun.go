package noun

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Native-Planet/go-bitstream"
	"github.com/spaolacci/murmur3"
	"math/big"
	"math/bits"
)

type Noun interface{}

type Cell struct {
	Head Noun
	Tail Noun
	Mug  uint32
}

func byteLength(i uint64) int {
	return (bits.Len64(i) + 7) / 8
}

func intBytes(i uint64) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, i)
	return buf.Bytes()[:byteLength(i)]
}

func mum(syd uint32, fal uint32, key uint64) uint32 {
	k := intBytes(key)
	for s := syd; s < syd+8; s++ {
		haz := murmur3.Sum32WithSeed(k, s)
		ham := (haz >> 31) ^ (haz & 0x7FFFFFFF)
		if ham != 0 {
			return ham
		}
	}
	return fal
}

func mugBoth(one uint32, two uint32) uint32 {
	return mum(0xdeadbeef, 0xfffe, uint64(two)<<32|uint64(one))
}

func deep(n Noun) bool {
	_, ok := n.(Cell)
	return ok
}

func (c *Cell) Hash() int {
	if c.Mug == 0 {
		c.Mug = mugBoth(uint32(mug(c.Head)), uint32(mug(c.Tail)))
	}
	return int(c.Mug)
}

func mug(n Noun) uint64 {
	if deep(n) {
		switch v := n.(type) {
		case Cell:
			return uint64(v.Hash())
		case *Cell:
			return uint64(v.Hash())
		}
	}
	switch v := n.(type) {
	case uint64:
		return uint64(mum(0xcafebabe, 0x7fff, v))
	case int:
		return uint64(mum(0xcafebabe, 0x7fff, uint64(v)))
	default:
		// Handle other types or panic with a more descriptive message
		panic(fmt.Sprintf("Unsupported type for mug: %T", n))
	}
} // ... [previous code]

func pretty(n Noun, tailPos bool) string {
	if deep(n) {
		cell := n.(Cell)
		content := fmt.Sprintf("%s %s", pretty(cell.Head, false), pretty(cell.Tail, true))
		if tailPos {
			return content
		}
		return "[" + content + "]"
	}
	return fmt.Sprintf("%d", n)
}

func translate(seq []Noun) Noun {
	if len(seq) == 0 {
		return uint64(0)
	}
	tail := seq[len(seq)-1]
	for i := len(seq) - 2; i >= 0; i-- {
		tail = Cell{seq[i], tail, 0}
	}
	return tail
}

func jamToStream(n Noun, out *bytes.Buffer) {
	cur := 0
	refs := make(map[Noun]int)

	bit := func(b bool) {
		if b {
			out.WriteByte(1)
		} else {
			out.WriteByte(0)
		}
		cur++
	}

	zero := func() {
		bit(false)
	}

	one := func() {
		bit(true)
	}

	rBits := func(num uint64, count int) {
		for i := 0; i < count; i++ {
			bit(0 != (num & (1 << i)))
		}
	}

	save := func(a Noun) {
		refs[a] = cur
	}

	mat := func(i int) {
		if i == 0 {
			one()
		} else {
			a := bits.Len(uint(i))
			b := bits.Len(uint(a))
			above := b + 1
			below := b - 1

			rBits(uint64(1<<b), above)
			rBits(uint64(a&((1<<below)-1)), below)
			rBits(uint64(i), a)
		}
	}

	back := func(ref int) {
		one()
		one()
		mat(ref)
	}

	var r func(a Noun)
	r = func(a Noun) {
		dupe, exists := refs[a]
		if deep(a) {
			if exists {
				back(dupe)
			} else {
				save(a)
				one()
				zero()
				cell := a.(Cell)
				r(cell.Head)
				r(cell.Tail)
			}
		} else if exists {
			isize := bits.Len(uint(a.(int)))
			dsize := bits.Len(uint(dupe))
			if isize < dsize {
				zero()
				mat(a.(int))
			} else {
				back(dupe)
			}
		} else {
			save(a)
			zero()
			mat(a.(int))
		}
	}
	r(n)
}

func cueFromStream(s *bitstream.BitReader) Noun {
	refs := make(map[int]Noun)
	var cur int

	readBits := func(n int) uint64 {
		result := 0
		for i := 0; i < n; i++ {
			bit, err := s.ReadBit()
			if err != nil {
				panic("oh shit")
			}
			if bit == bitstream.One {
				result |= 1 << i
			} else {
				result |= 0 << i
			}
		}
		return uint64(result)
	}

	one := func() bool {
		val, _ := s.ReadBit()
		cur++
		return val == bitstream.One

	}

	rub := func() uint64 {
		z := 0
		for !one() {
			z++
		}
		if z == 0 {
			return 0
		}

		below := z - 1
		lbits := readBits(below)

		bex := uint64(1 << below)
		val := readBits(int(bex ^ lbits))
		return val
	}

	var r func(start int) (Noun, int)
	r = func(start int) (Noun, int) {
		var ret Noun
		if one() {
			if one() {
				refValue := int(rub())
				ret = refs[refValue]
			} else {
				head, newCur := r(cur)
				tail, newCur := r(newCur)
				ret = Cell{head, tail, 0}
				cur = newCur
			}
		} else {
			ret = rub()
		}
		refs[start] = ret
		return ret, cur
	}
	ret, _ := r(cur)
	return ret
}

func readInt(length int, s *bytes.Buffer) *big.Int {
	var r big.Int

	for i := 0; i < length; i++ {
		val, _ := s.ReadByte()
		if val == 1 {
			// Shift the bit to the left by the current position and add it to r
			r.SetBit(&r, i, 1)
		}
	}

	return &r
}

func jam(n Noun) *big.Int {
	out := new(bytes.Buffer)
	jamToStream(n, out)
	//fmt.Println(out.Bytes())
	return readInt(out.Len(), out)
}

func cue(i *big.Int) Noun {
	var buf bytes.Buffer
	w := bitstream.NewWriter(&buf)

	zero := big.NewInt(0)
	one := big.NewInt(1)
	bitLen := i.BitLen() % 8

	for j := 0; j < bitLen; j++ {
		if err := w.WriteBit(bitstream.Zero); err != nil {
			panic("Failed to write bit")
		}
	}

	for i.Cmp(zero) > 0 { // i > 0
		bit := new(big.Int).And(i, one) // i & 1

		bitValue := uint8(bit.Uint64())

		if bitValue == 0 {
			if err := w.WriteBit(bitstream.Zero); err != nil {
				panic("Failed to write bit")
			}
		} else {
			if err := w.WriteBit(bitstream.One); err != nil {
				panic("Failed to write bit")
			}

		}

		i = i.Rsh(i, 1) // i >>= 1
	}

	if err := w.Flush(bitstream.Zero); err != nil {
		panic("Failed to flush writer")
	}

	r := bitstream.NewReader(&buf)
	for j := 0; j < bitLen; j++ {
		r.ReadBit()
	}

	return cueFromStream(r)
}

// func main() {
/*
	// Test byteLength
	fmt.Println(byteLength(0))                 // Expected: 0
	fmt.Println(byteLength(255))               // Expected: 1
	fmt.Println(byteLength(256))               // Expected: 2

	// Test intBytes
	fmt.Printf("%x\n", intBytes(0))            // Expected: ''
	fmt.Printf("%x\n", intBytes(256))          // Expected: '0001'

	// Test mum
	fmt.Println(mum(0xcafebabe, 0x7fff, 0))    // Expected: 2046756072 (or similar based on hash function)
	fmt.Println(mum(0xdeadbeef, 0xfffe, 8790750394176177384)) // Expected: 422532488 (or similar based on hash function)

	// Test mugBoth
	fmt.Println(mugBoth(2046756072, 2046756072)) // Expected: 422532488 (or similar based on hash function)

	// Test deep
	fmt.Println(deep(1))                       // Expected: false
	fmt.Println(deep(Cell{1, 2, 0}))           // Expected: true

	// Test mug
	fmt.Println(mug(0))                        // Expected: 2046756072 (or similar based on hash function)
	fmt.Println(mug(Cell{0, 0, 0}))            // Expected: 422532488 (or similar based on hash function)

  fmt.Println("wtf")*/
//nn := cue(3426417)
//fmt.Println(pretty(nn, false))
//fmt.Println(jam(0))          // Expected: [1 [2 3]]
//fmt.Println(jam(1))          // Expected: [1 [2 3]]
//fmt.Println(jam(1234567890987654321))
//fmt.Println(jam(Cell{0, 0, 0}))          // Expected: [1 [2 3]]

//   i := new(big.Int)
//   i.SetString("1569560238373119425266963811040232206341",10)
// 	fmt.Println(pretty(cue(i),false))

//jtest := Cell{Cell{1234567890987654321, 1234567890987654321, 0}, Cell{1234567890987654321, 1234567890987654321, 0}, 0}
//fmt.Println(jam(jtest))
//fmt.Println(pretty(cue(jam(jtest)), false))

// Test pretty
/*
	noun1 := Cell{1, Cell{2, 3, 0}, 0}
	fmt.Println(pretty(noun1, false))          // Expected: [1 [2 3]]

	// Test translate
	seq := []Noun{1, Cell{2, 3, 0}, 4}
	noun2 := translate(seq)
	fmt.Println(pretty(noun2, false))          // Expected: [1 [2 3] 4]

	// Test jam and cue
	j := jam(noun1)
	fmt.Println(j)
	c := cue(j)
	fmt.Println(pretty(c, false))              // Expected: [1 [2 3]]

	// Additional tests based on the original Python doctests
	x := Cell{1, Cell{2, 3, 0}, 0}
	fmt.Println(x.Head)                        // Expected: 1
	tail := x.Tail.(Cell)
	fmt.Println(tail.Head)                     // Expected: 2
	fmt.Println(tail.Tail)                     // Expected: 3

	y := Cell{Cell{1, 2, 0}, Cell{3, 4, 0}, 0}
	fmt.Println(mug(y))                        // Expected: 1496649457 (or similar based on hash function)
	// Note: The following tests related to reference equality in Python do not have a direct equivalent in Go.
	// In Go, we don't have the concept of sharing storage for struct types like in Python.
	// So, we'll skip the tests that check for reference equality.

	// Test pretty function with different tail positions
	x = Cell{0, 0, 0}
	fmt.Println(pretty(x, false))              // Expected: [0 0]
	fmt.Println(pretty(x, true))               // Expected: 0 0
*/
// }
