package ob

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"unsafe"

	"github.com/spaolacci/murmur3"
	"lukechampine.com/urbit/atom"
)

type AzimuthPoint uint32

func (p AzimuthPoint) IsGalaxy() bool { return p < 256 }
func (p AzimuthPoint) IsStar() bool   { return 256 <= p && p < 65536 }
func (p AzimuthPoint) IsPlanet() bool { return 65536 <= p }

func (p AzimuthPoint) ChildStar(i uint8) AzimuthPoint {
	if !p.IsGalaxy() {
		panic("only galaxies can spawn stars")
	} else if i == 0 {
		panic("child index must be greater than 0")
	}
	return p | (AzimuthPoint(i) << 8)
}

func (p AzimuthPoint) ChildPlanet(i uint16) AzimuthPoint {
	if !p.IsStar() {
		panic("only stars can spawn planets")
	} else if i == 0 {
		panic("child index must be greater than 0")
	}
	return p | (AzimuthPoint(i) << 16)
}

func (p AzimuthPoint) Parent() AzimuthPoint {
	switch {
	case p.IsGalaxy():
		panic("galaxies do not have parents")
	case p.IsStar():
		return p & 0x000000FF
	case p.IsPlanet():
		return p & 0x0000FFFF
	}
	panic("unreachable")
}

func (p AzimuthPoint) String() string {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, fein(p))
	return atom.FromBytes(buf).Format("p")
}

func PointFromName(n string) (AzimuthPoint, error) {
	buf := make([]byte, 4)
	ok := []bool{true, true, true, true}
	switch len(n) {
	case 14: // ~dopzod-dopzod
		buf[0], ok[0] = phonemeIndex[n[1:4]]
		buf[1], ok[1] = phonemeIndex[n[4:7]]
		n = n[7:]
		fallthrough
	case 7: // ~dopzod
		buf[2], ok[2] = phonemeIndex[n[1:4]]
		n = n[3:]
		fallthrough
	case 4: // ~zod
		buf[3], ok[3] = phonemeIndex[n[1:4]]
	default:
		return 0, errors.New("invalid length")
	}
	if !ok[0] || !ok[1] || !ok[2] || !ok[3] {
		return 0, errors.New("invalid phoneme")
	}

	return fynd(binary.BigEndian.Uint32(buf)), nil
}

type Comet [16]byte

func (c Comet) String() string {
	parts := make([]interface{}, 8)
	for i := range parts {
		parts[i] = AzimuthPoint(binary.BigEndian.Uint16(c[i*2:])).String()[1:]
	}
	return fmt.Sprintf("~%v-%v-%v-%v--%v-%v-%v-%v", parts...)
}

func (c Comet) Parent() AzimuthPoint {
	return AzimuthPoint(binary.BigEndian.Uint16(c[14:]))
}

func FindComet(star AzimuthPoint) (Comet, string) {
	if !star.IsStar() {
		panic("not a star")
	}
	seed := make([]byte, 32)
	rand.Read(seed)
	pubbuf, secbuf := [65]byte{'b'}, [65]byte{'B'}
	for ; ; *(*uint64)(unsafe.Pointer(&seed[0]))++ {
		// derive keypair
		bits := sha512.Sum512(seed)
		cry := ed25519.NewKeyFromSeed(bits[:32])
		sgn := ed25519.NewKeyFromSeed(bits[32:])
		pub := append(append(pubbuf[:1], cry[32:]...), sgn[32:]...)
		sec := append(append(secbuf[:1], cry[:32]...), sgn[:32]...)

		// fingerprint
		pubsum := sha256.Sum256(pub)
		*(*uint32)(unsafe.Pointer(&pubsum)) ^= 0x67696662
		h := sha256.Sum256(pubsum[:])
		var c Comet
		for i := range c {
			c[15-i] = h[i] ^ h[16+i]
		}

		if c.Parent() == star {
			return c, atom.FromBytes(jamComet(c, sec)).Format("uw")
		}
	}
}

func jamComet(who Comet, key []byte) []byte {
	// This code is truly shameful.
	// Please do not look at it.

	mat := func(a []byte) string {
		var buf bytes.Buffer
		for _, b := range a {
			fmt.Fprintf(&buf, "%08b", b)
		}
		s := strings.TrimLeft(buf.String(), "0")
		switch s {
		case "":
			return "10"
		case "1":
			return "1100"
		default:
			met := bits.Len16(uint16(len(s))) - 1
			return s + fmt.Sprintf("%0[1]*b%08b", met, len(s)%(1<<met), 4<<met)
		}
	}
	jam := func(elems ...[]byte) string {
		var sb strings.Builder
		sb.WriteString(mat(elems[0]))
		for i := 1; i < len(elems); i++ {
			sb.WriteString(mat(elems[i]))
			sb.WriteString("01")
		}
		return sb.String()
	}

	// flip key endianness
	flippedKey := make([]byte, len(key))
	for i := range flippedKey {
		flippedKey[i] = key[len(key)-i-1]
	}

	// jam to binary, pad to byte-boundary, decode to bytes
	bits := jam(nil, flippedKey, []byte{1}, who[:])
	for len(bits)%8 != 0 {
		bits = "0" + bits
	}
	dec := make([]byte, len(bits)/8)
	for i := range dec {
		b, _ := strconv.ParseUint(bits[i*8:][:8], 2, 8)
		dec[i] = byte(b)
	}
	return dec
}

func prf(j int, u uint16) uint32 {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, u)
	return murmur3.Sum32WithSeed(data, []uint32{
		0xb76d5eed,
		0xee281300,
		0x85bcae01,
		0x4b387af7,
	}[j])
}

func fein(p AzimuthPoint) uint32 {
	if !p.IsPlanet() {
		return uint32(p)
	}
	const a = 65535
	const b = 65536
	m := uint32(p) - b
	l, r := m%a, m/a
	for j := 0; j < 4; j++ {
		tmp := uint64(l) + uint64(prf(j, uint16(r)))
		if j%2 == 0 {
			tmp %= a
		} else {
			tmp %= b
		}
		l, r = r, uint32(tmp)
	}
	if r == a {
		// special handling for collisions
		return b + a*r + l
	}
	return b + a*l + r
}

func fynd(arg uint32) AzimuthPoint {
	if !AzimuthPoint(arg).IsPlanet() {
		return AzimuthPoint(arg)
	}
	const a = 65535
	const b = 65536
	m := arg - b
	l := m % a
	r := m / a
	if r != a {
		l, r = r, l
	}
	for j := 4; j > 0; j-- {
		eff := prf(j-1, uint16(l))
		var tmp uint32
		if j%2 != 0 {
			tmp = (r + a - eff%a) % a
		} else {
			tmp = (r + b - eff%b) % b
		}
		l, r = tmp, l
	}
	return AzimuthPoint(65536 + a*r + l)
}

var phonemeIndex = func() map[string]uint8 {
	m := make(map[string]uint8)
	for i, p := range prefixes {
		m[p] = uint8(i)
	}
	for i, p := range suffixes {
		m[p] = uint8(i)
	}
	return m
}()

var prefixes = [256]string{
	"doz", "mar", "bin", "wan", "sam", "lit", "sig", "hid", "fid", "lis", "sog", "dir", "wac", "sab", "wis", "sib",
	"rig", "sol", "dop", "mod", "fog", "lid", "hop", "dar", "dor", "lor", "hod", "fol", "rin", "tog", "sil", "mir",
	"hol", "pas", "lac", "rov", "liv", "dal", "sat", "lib", "tab", "han", "tic", "pid", "tor", "bol", "fos", "dot",
	"los", "dil", "for", "pil", "ram", "tir", "win", "tad", "bic", "dif", "roc", "wid", "bis", "das", "mid", "lop",
	"ril", "nar", "dap", "mol", "san", "loc", "nov", "sit", "nid", "tip", "sic", "rop", "wit", "nat", "pan", "min",
	"rit", "pod", "mot", "tam", "tol", "sav", "pos", "nap", "nop", "som", "fin", "fon", "ban", "mor", "wor", "sip",
	"ron", "nor", "bot", "wic", "soc", "wat", "dol", "mag", "pic", "dav", "bid", "bal", "tim", "tas", "mal", "lig",
	"siv", "tag", "pad", "sal", "div", "dac", "tan", "sid", "fab", "tar", "mon", "ran", "nis", "wol", "mis", "pal",
	"las", "dis", "map", "rab", "tob", "rol", "lat", "lon", "nod", "nav", "fig", "nom", "nib", "pag", "sop", "ral",
	"bil", "had", "doc", "rid", "moc", "pac", "rav", "rip", "fal", "tod", "til", "tin", "hap", "mic", "fan", "pat",
	"tac", "lab", "mog", "sim", "son", "pin", "lom", "ric", "tap", "fir", "has", "bos", "bat", "poc", "hac", "tid",
	"hav", "sap", "lin", "dib", "hos", "dab", "bit", "bar", "rac", "par", "lod", "dos", "bor", "toc", "hil", "mac",
	"tom", "dig", "fil", "fas", "mit", "hob", "har", "mig", "hin", "rad", "mas", "hal", "rag", "lag", "fad", "top",
	"mop", "hab", "nil", "nos", "mil", "fop", "fam", "dat", "nol", "din", "hat", "nac", "ris", "fot", "rib", "hoc",
	"nim", "lar", "fit", "wal", "rap", "sar", "nal", "mos", "lan", "don", "dan", "lad", "dov", "riv", "bac", "pol",
	"lap", "tal", "pit", "nam", "bon", "ros", "ton", "fod", "pon", "sov", "noc", "sor", "lav", "mat", "mip", "fip",
}

var suffixes = [256]string{
	"zod", "nec", "bud", "wes", "sev", "per", "sut", "let", "ful", "pen", "syt", "dur", "wep", "ser", "wyl", "sun",
	"ryp", "syx", "dyr", "nup", "heb", "peg", "lup", "dep", "dys", "put", "lug", "hec", "ryt", "tyv", "syd", "nex",
	"lun", "mep", "lut", "sep", "pes", "del", "sul", "ped", "tem", "led", "tul", "met", "wen", "byn", "hex", "feb",
	"pyl", "dul", "het", "mev", "rut", "tyl", "wyd", "tep", "bes", "dex", "sef", "wyc", "bur", "der", "nep", "pur",
	"rys", "reb", "den", "nut", "sub", "pet", "rul", "syn", "reg", "tyd", "sup", "sem", "wyn", "rec", "meg", "net",
	"sec", "mul", "nym", "tev", "web", "sum", "mut", "nyx", "rex", "teb", "fus", "hep", "ben", "mus", "wyx", "sym",
	"sel", "ruc", "dec", "wex", "syr", "wet", "dyl", "myn", "mes", "det", "bet", "bel", "tux", "tug", "myr", "pel",
	"syp", "ter", "meb", "set", "dut", "deg", "tex", "sur", "fel", "tud", "nux", "rux", "ren", "wyt", "nub", "med",
	"lyt", "dus", "neb", "rum", "tyn", "seg", "lyx", "pun", "res", "red", "fun", "rev", "ref", "mec", "ted", "rus",
	"bex", "leb", "dux", "ryn", "num", "pyx", "ryg", "ryx", "fep", "tyr", "tus", "tyc", "leg", "nem", "fer", "mer",
	"ten", "lus", "nus", "syl", "tec", "mex", "pub", "rym", "tuc", "fyl", "lep", "deb", "ber", "mug", "hut", "tun",
	"byl", "sud", "pem", "dev", "lur", "def", "bus", "bep", "run", "mel", "pex", "dyt", "byt", "typ", "lev", "myl",
	"wed", "duc", "fur", "fex", "nul", "luc", "len", "ner", "lex", "rup", "ned", "lec", "ryd", "lyd", "fen", "wel",
	"nyd", "hus", "rel", "rud", "nes", "hes", "fet", "des", "ret", "dun", "ler", "nyr", "seb", "hul", "ryl", "lud",
	"rem", "lys", "fyn", "wer", "ryc", "sug", "nys", "nyl", "lyn", "dyn", "dem", "lux", "fed", "sed", "bec", "mun",
	"lyr", "tes", "mud", "nyt", "byr", "sen", "weg", "fyr", "mur", "tel", "rep", "teg", "pec", "nel", "nev", "fes",
}
