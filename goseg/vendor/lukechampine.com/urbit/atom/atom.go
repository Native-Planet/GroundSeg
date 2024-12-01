package atom

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// An Aura is a type hint that controls how an Atom is printed.
type Aura string

// Atom auras.
const (
	AuraAtom = ""    // no aura
	AuraD    = "d"   // date
	AuraDA   = "da"  // absolute date
	AuraDR   = "dr"  // relative date
	AuraP    = "p"   // phonemic base (ship name)
	AuraR    = "r"   // IEEE floating-point
	AuraRD   = "rd"  // double precision  (64 bits)
	AuraRH   = "rh"  // half precision (16 bits)
	AuraRQ   = "rq"  // quad precision (128 bits)
	AuraRS   = "rs"  // single precision (32 bits)
	AuraS    = "s"   // signed integer, sign bit low
	AuraSB   = "sb"  // signed binary
	AuraSD   = "sd"  // signed decimal
	AuraSV   = "sv"  // signed base32
	AuraSW   = "sw"  // signed base64
	AuraSX   = "sx"  // signed hexadecimal
	AuraT    = "t"   // UTF-8 text (cord)
	AuraTA   = "ta"  // ASCII text (knot)
	AuraTAS  = "tas" // ASCII text symbol (term)
	AuraU    = "u"   // unsigned integer
	AuraUB   = "ub"  // unsigned binary
	AuraUD   = "ud"  // unsigned decimal
	AuraUV   = "uv"  // unsigned base32
	AuraUW   = "uw"  // unsigned base64
	AuraUX   = "ux"  // unsigned hexadecimal
)

// NestsIn returns whether b nests in a.
func (a Aura) NestsIn(b Aura) bool {
	return len(a) >= len(b) && a[:len(b)] == b
}

// An Atom is a natural number.
type Atom struct {
	i    *big.Int
	aura Aura
}

var uwEnc = base64.NewEncoding("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-~")

// String implements fmt.Stringer, rendering the Atom according to its aura.
func (a Atom) String() string {
	if a.aura.NestsIn("s") {
		n := new(big.Int).Add(a.i, big.NewInt(1))
		u := Atom{n.Rsh(n, 1), "u" + a.aura[1:]}
		return "--"[a.i.Bit(0):] + u.String()
	}

	switch a.aura {
	default:
		panic("unsupported aura")
	case AuraP:
		return formatP(a.i)
	case AuraDA:
		return "~" + formatDate(a.i)
	case AuraDR:
		return "~" + formatDuration(a.i)
	case AuraD, AuraR:
		return "0x" + a.i.Text(16)
	case AuraRD:
		return ".~" + formatFloat(math.Float64frombits(a.i.Uint64()), 64)
	case AuraRH:
		return ".~~" + float16(uint16(a.i.Uint64()))
	case AuraRS:
		return "." + formatFloat(float64(math.Float32frombits(uint32(a.i.Uint64()))), 32)
	case AuraRQ:
		return ".~~~" + float128(a.i)
	case AuraT:
		return "'" + string(flip(a.i.Bytes())) + "'"
	case AuraTA:
		return "~." + string(flip(a.i.Bytes()))
	case AuraTAS:
		if a.i.BitLen() == 0 {
			return "%$"
		}
		return "%" + string(flip(a.i.Bytes()))
	case AuraAtom, AuraU, AuraUD:
		return formatInt(a.i.Text(10), 3)
	case AuraUB:
		return "0b" + formatInt(a.i.Text(2), 4)
	case AuraUV:
		return "0v" + formatInt(a.i.Text(32), 5)
	case AuraUW:
		if a.i.BitLen() == 0 {
			return "0w0"
		}
		return "0w" + formatInt(uwEnc.EncodeToString(pad(a.i.Bytes(), 3)), 5)
	case AuraUX:
		return "0x" + formatInt(a.i.Text(16), 4)
	}
}

// Cast returns a copy of a with the specified aura.
func (a Atom) Cast(aura Aura) Atom {
	return Atom{
		i:    new(big.Int).Set(a.i),
		aura: aura,
	}
}

// Format is shorthand for a.Cast(aura).String().
func (a Atom) Format(aura Aura) string {
	return a.Cast(aura).String()
}

// New initializes an Atom with a *big.Int.
func New(i *big.Int) Atom {
	return Atom{
		i:    new(big.Int).Set(i),
		aura: AuraAtom,
	}
}

// New64 initializes an Atom with a uint64.
func New64(u uint64) Atom {
	return New(new(big.Int).SetUint64(u))
}

// FromBytes parses an Atom from a []byte.
func FromBytes(b []byte) Atom {
	return New(new(big.Int).SetBytes(b))
}

func pad(b []byte, n int) []byte {
	for len(b)%n != 0 {
		b = append([]byte{0}, b...)
	}
	return b
}

func flip(b []byte) []byte {
	for i := range b[:len(b)/2] {
		j := len(b) - i - 1
		b[i], b[j] = b[j], b[i]
	}
	return b
}

func formatP(a *big.Int) string {
	var b []byte
	b = a.Bytes()
	if len(b) == 0 {
		b = []byte{0}
	}
	if len(b) > 1 {
		b = pad(b, 2)
	}
	var buf strings.Builder
	buf.Grow(len(b)*3 + len(b)/2 + len(b)/8)
	buf.WriteByte('~')
	for i, c := range b {
		j := len(b) - i
		if i > 0 && j%2 == 0 {
			buf.WriteByte('-')
			if j%8 == 0 {
				buf.WriteByte('-')
			}
		}
		if j%2 == 0 {
			buf.WriteString(prefixes[c])
		} else {
			buf.WriteString(suffixes[c])
		}
	}
	return buf.String()
}

func formatFloat(f float64, bits int) string {
	s := strings.ToLower(strconv.FormatFloat(f, 'e', -1, bits))
	if !strings.Contains(s, "e") {
		return strings.TrimPrefix(s, "+") // inf or nan
	}
	t := strings.Split(s, "e")
	frac := t[0]
	sign := strings.TrimPrefix(t[1][:1], "+")
	exp := strings.TrimLeft(t[1][1:], "0")
	return strings.TrimSuffix(frac+"e"+sign+exp, "e")
}

func float16(bits uint16) string {
	sign := ""
	if bits&0x8000 != 0 {
		sign = "-"
	}
	exp := int(bits & 0x7C00 >> 10)
	frac := bits & 0x03FF
	x := new(big.Float).SetPrec(11)
	lead := 1
	exponent := exp - 15
	switch exp {
	case 0x1F:
		if frac == 0 {
			return sign + "inf"
		}
		return "nan"
	case 0x00:
		if frac == 0 {
			return sign + "0"
		}
		lead = 0
		exponent = -14
	}
	x.Parse(fmt.Sprintf("%s0b%d.%010bp%d", sign, lead, frac, exponent), 0)
	t := strings.Split(x.Text('e', -1), "e")
	sign = strings.TrimLeft(t[1][:1], "+")
	t[1] = strings.TrimLeft(t[1][1:], "0")
	if t[1] == "" {
		return t[0]
	}
	return t[0] + "e" + sign + t[1]
}

func float128(i *big.Int) string {
	b := i.Uint64()
	a := new(big.Int).Rsh(i, 64).Uint64()

	sign := ""
	if a&0x8000000000000000 != 0 {
		sign = "-"
	}
	exp := int(a&0x7FFF000000000000) >> 48
	frac1, frac2 := a&0x0000FFFFFFFFFFFF, b
	x := new(big.Float).SetPrec(113)
	lead := 1
	exponent := exp - 16383
	switch exp {
	case 0x7FFF:
		if frac1 == 0 && frac2 == 0 {
			return sign + "inf"
		}
		return "nan"
	case 0x00:
		if frac1 == 0 && frac2 == 0 {
			return sign + "0"
		}
		lead = 0
		exponent = -16382
	}
	x.Parse(fmt.Sprintf("%s0b%d.%sp%d", sign, lead, fmt.Sprintf("%048b%064b", frac1, frac2), exponent), 0)
	t := strings.Split(x.Text('e', -1), "e")
	sign = strings.TrimLeft(t[1][:1], "+")
	t[1] = strings.TrimLeft(t[1][1:], "0")
	if t[1] == "" {
		return t[0]
	}
	return t[0] + "e" + sign + t[1]
}

var jesus = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
var oneSec = new(big.Int).Lsh(big.NewInt(1), 64)

func formatDate(a *big.Int) string {
	secs, rem := new(big.Int).QuoRem(a, oneSec, new(big.Int))
	date := time.Unix(jesus+int64(secs.Uint64())-9223372029693628800, 0).UTC().Format("2006.1.2..15.04.05")
	frac := formatInt("."+hex.EncodeToString(rem.Bytes()), 4)
	return strings.TrimRight(strings.TrimLeft(date, "0"), ".0") + strings.TrimRight(frac, ".0")
}

func formatDuration(a *big.Int) string {
	if a.BitLen() == 0 {
		return "s0"
	}
	secs, rem := new(big.Int).QuoRem(a, oneSec, new(big.Int))
	mins, secRem := new(big.Int).QuoRem(secs, big.NewInt(60), new(big.Int))
	hrs, minRem := new(big.Int).QuoRem(mins, big.NewInt(60), new(big.Int))
	days, hrRem := new(big.Int).QuoRem(hrs, big.NewInt(24), new(big.Int))
	var sb strings.Builder
	if days.BitLen() > 0 {
		sb.WriteString(".d" + days.String())
	}
	if hrRem.BitLen() > 0 {
		sb.WriteString(".h" + hrRem.String())
	}
	if minRem.BitLen() > 0 {
		sb.WriteString(".m" + minRem.String())
	}
	if secRem.BitLen() > 0 {
		sb.WriteString(".s" + secRem.String())
	}
	if rem.BitLen() > 0 {
		if sb.Len() == 0 {
			sb.WriteString("s0")
		}
		buf := []byte(".0000000000000000")
		rt := rem.Text(16)
		copy(buf[len(buf)-len(rt):], rt)
		sb.WriteString(formatInt(string(buf), 4))
	}
	return strings.TrimPrefix(sb.String(), ".")
}

func formatInt(s string, n int) string {
	if s == "0" {
		return "0"
	}
	s = strings.TrimLeft(s, "0")
	if len(s) < n {
		return s
	}
	i := len(s) % n
	if i == 0 {
		i = n
	}
	var buf strings.Builder
	buf.Grow(len(s) + len(s)/n)
	buf.WriteString(s[:i])
	for ; i < len(s); i += n {
		buf.WriteByte('.')
		buf.WriteString(s[i:][:n])
	}
	return buf.String()
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
