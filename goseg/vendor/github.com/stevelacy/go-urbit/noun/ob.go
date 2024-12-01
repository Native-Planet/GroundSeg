package noun

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var u_65535 *big.Int
var u_65536 *big.Int
var ux_100 *big.Int
var ux_ffff_ffff *big.Int
var ux_1_0000_0000 *big.Int
var ux_ffff_ffff_ffff_ffff *big.Int
var ux_ffff_ffff_0000_0000 *big.Int

var prefixes = [256]string{"doz", "mar", "bin", "wan", "sam", "lit", "sig", "hid", "fid", "lis", "sog", "dir", "wac", "sab", "wis", "sib", "rig", "sol", "dop", "mod", "fog", "lid", "hop", "dar", "dor", "lor", "hod", "fol", "rin", "tog", "sil", "mir", "hol", "pas", "lac", "rov", "liv", "dal", "sat", "lib", "tab", "han", "tic", "pid", "tor", "bol", "fos", "dot", "los", "dil", "for", "pil", "ram", "tir", "win", "tad", "bic", "dif", "roc", "wid", "bis", "das", "mid", "lop", "ril", "nar", "dap", "mol", "san", "loc", "nov", "sit", "nid", "tip", "sic", "rop", "wit", "nat", "pan", "min", "rit", "pod", "mot", "tam", "tol", "sav", "pos", "nap", "nop", "som", "fin", "fon", "ban", "mor", "wor", "sip", "ron", "nor", "bot", "wic", "soc", "wat", "dol", "mag", "pic", "dav", "bid", "bal", "tim", "tas", "mal", "lig", "siv", "tag", "pad", "sal", "div", "dac", "tan", "sid", "fab", "tar", "mon", "ran", "nis", "wol", "mis", "pal", "las", "dis", "map", "rab", "tob", "rol", "lat", "lon", "nod", "nav", "fig", "nom", "nib", "pag", "sop", "ral", "bil", "had", "doc", "rid", "moc", "pac", "rav", "rip", "fal", "tod", "til", "tin", "hap", "mic", "fan", "pat", "tac", "lab", "mog", "sim", "son", "pin", "lom", "ric", "tap", "fir", "has", "bos", "bat", "poc", "hac", "tid", "hav", "sap", "lin", "dib", "hos", "dab", "bit", "bar", "rac", "par", "lod", "dos", "bor", "toc", "hil", "mac", "tom", "dig", "fil", "fas", "mit", "hob", "har", "mig", "hin", "rad", "mas", "hal", "rag", "lag", "fad", "top", "mop", "hab", "nil", "nos", "mil", "fop", "fam", "dat", "nol", "din", "hat", "nac", "ris", "fot", "rib", "hoc", "nim", "lar", "fit", "wal", "rap", "sar", "nal", "mos", "lan", "don", "dan", "lad", "dov", "riv", "bac", "pol", "lap", "tal", "pit", "nam", "bon", "ros", "ton", "fod", "pon", "sov", "noc", "sor", "lav", "mat", "mip", "fip"}

var suffixes = [256]string{"zod", "nec", "bud", "wes", "sev", "per", "sut", "let", "ful", "pen", "syt", "dur", "wep", "ser", "wyl", "sun", "ryp", "syx", "dyr", "nup", "heb", "peg", "lup", "dep", "dys", "put", "lug", "hec", "ryt", "tyv", "syd", "nex", "lun", "mep", "lut", "sep", "pes", "del", "sul", "ped", "tem", "led", "tul", "met", "wen", "byn", "hex", "feb", "pyl", "dul", "het", "mev", "rut", "tyl", "wyd", "tep", "bes", "dex", "sef", "wyc", "bur", "der", "nep", "pur", "rys", "reb", "den", "nut", "sub", "pet", "rul", "syn", "reg", "tyd", "sup", "sem", "wyn", "rec", "meg", "net", "sec", "mul", "nym", "tev", "web", "sum", "mut", "nyx", "rex", "teb", "fus", "hep", "ben", "mus", "wyx", "sym", "sel", "ruc", "dec", "wex", "syr", "wet", "dyl", "myn", "mes", "det", "bet", "bel", "tux", "tug", "myr", "pel", "syp", "ter", "meb", "set", "dut", "deg", "tex", "sur", "fel", "tud", "nux", "rux", "ren", "wyt", "nub", "med", "lyt", "dus", "neb", "rum", "tyn", "seg", "lyx", "pun", "res", "red", "fun", "rev", "ref", "mec", "ted", "rus", "bex", "leb", "dux", "ryn", "num", "pyx", "ryg", "ryx", "fep", "tyr", "tus", "tyc", "leg", "nem", "fer", "mer", "ten", "lus", "nus", "syl", "tec", "mex", "pub", "rym", "tuc", "fyl", "lep", "deb", "ber", "mug", "hut", "tun", "byl", "sud", "pem", "dev", "lur", "def", "bus", "bep", "run", "mel", "pex", "dyt", "byt", "typ", "lev", "myl", "wed", "duc", "fur", "fex", "nul", "luc", "len", "ner", "lex", "rup", "ned", "lec", "ryd", "lyd", "fen", "wel", "nyd", "hus", "rel", "rud", "nes", "hes", "fet", "des", "ret", "dun", "ler", "nyr", "seb", "hul", "ryl", "lud", "rem", "lys", "fyn", "wer", "ryc", "sug", "nys", "nyl", "lyn", "dyn", "dem", "lux", "fed", "sed", "bec", "mun", "lyr", "tes", "mud", "nyt", "byr", "sen", "weg", "fyr", "mur", "tel", "rep", "teg", "pec", "nel", "nev", "fes"}

func init() {
	// initialize magic numbers to big ints
	u_65535 = B(65535)
	u_65536 = B(65536)
	ux_100 = B(0x100)
	ux_ffff_ffff = B(0xffffffff)
	ux_1_0000_0000 = B(0)
	ux_1_0000_0000.SetString("100000000", 16)

	ux_ffff_ffff_ffff_ffff = B(0)
	fmt.Sscan("0xffffffffffffffff", ux_ffff_ffff_ffff_ffff)

	ux_ffff_ffff_0000_0000 = B(0)
	fmt.Sscan("0xffffffff00000000", ux_ffff_ffff_0000_0000)
}

// Patp2hex converts a patp (~zod) to the hex value (0)
func Patp2hex(name string) (string, error) {
	bn, err := Patp2bn(name)
	if err != nil {
		return "", err
	}
	return bn.Text(16), nil
}

func Bex(n *big.Int) *big.Int {
	return B(0).Exp(B(2), n, nil)
}

func rsh(a, b, c *big.Int) *big.Int {
	be := Bex(a)
	return B(0).Div(c, Bex(B(0).Mul(be, b)))
}

func end(a, b, c *big.Int) *big.Int {
	be := Bex(a)
	return B(0).Mod(c, Bex(be.Mul(be, b)))
}

func met(a, b, c *big.Int) *big.Int {
	if b.Cmp(B(0)) == 0 {
		return c
	}
	return met(a, rsh(a, B(1), b), B(0).Add(c, B(1)))
}

// Clan returns the ship class of a patp
func Clan(name string) (string, error) {
	p, err := Patp2bn(name)
	if err != nil {
		return "", err
	}
	wid := met(B(3), p, B(0))
	if wid.Cmp(B(1)) < 0 {
		return "galaxy", nil
	}
	if wid.Cmp(B(2)) == 0 {
		return "star", nil
	}
	if wid.Cmp(B(4)) < 0 {
		return "planet", nil
	}
	if wid.Cmp(B(8)) < 0 || wid.Cmp(B(8)) == 0 {
		return "moon", nil
	}
	return "comet", nil
}

// Sein returns the parent patp as a big.Int
func Sein(name string) (*big.Int, error) {
	p, err := Patp2bn(name)
	if err != nil {
		return B(0), err
	}
	clan, err := Clan(name)
	if err != nil {
		return B(0), err
	}
	switch clan {
	case "galaxy":
		return p, nil
	case "star":
		return end(B(3), B(1), p), nil
	case "planet":
		return end(B(4), B(1), p), nil
	case "moon":
		return end(B(5), B(1), p), nil

	default:
		{
			return B(0), nil
		}
	}
}

// BN2patp turns a patp big.Int into the string form
func BN2patp(name *big.Int) (string, error) {
	return Hex2patp(name.Text(16))
}

// Hex2patp converts the hex (ec) to a patp (~fed)
func Hex2patp(hex string) (string, error) {
	bn := B(0)
	bn, _ = bn.SetString(hex, 16)
	sxz := Fynd(bn, feis)
	dyy := met(B(4), sxz, B(0))

	var loop func(tsxz, timp *big.Int, trep string) string
	loop = func(tsxz, timp *big.Int, trep string) string {
		log := end(B(4), B(1), tsxz)
		pre := prefixes[rsh(B(3), B(1), log).Int64()]
		suf := suffixes[end(B(3), B(1), log).Int64()]

		etc := "-"
		if B(0).Mod(timp, B(4)).Cmp(B(0)) == 0 {
			if timp.Cmp(B(0)) == 0 {
				etc = ""
			} else {
				etc = "--"
			}
		}

		res := pre + suf + etc + trep
		if timp.Cmp(dyy) == 0 {
			return trep
		}
		ti := B(0).Add(timp, B(1))
		return loop(rsh(B(4), B(1), tsxz), ti, res)
	}

	dyx := met(B(3), sxz, B(0))

	tmp := ""
	if dyx.Cmp(B(0)) == 0 {
		tmp = suffixes[0]
	} else if dyx.Cmp(B(1)) == 0 {
		tmp = suffixes[sxz.Int64()]
	} else {
		tmp = loop(sxz, B(0), "")
	}
	return "~" + tmp, nil
}

// Patp2bn converts a patp to a big int
func Patp2bn(name string) (*big.Int, error) {
	if !isValidPat(name) {
		return nil, fmt.Errorf("invalid name %s", name)
	}
	addr := makeAddr(name)
	bn := Fynd(addr, tail)
	return bn, nil
}

func Fynd(bn *big.Int, fn func(*big.Int) *big.Int) *big.Int {
	lo := B(0).And(bn, ux_ffff_ffff)
	hi := B(0).And(bn, ux_ffff_ffff_0000_0000)

	if bn.Cmp(u_65536) > 0 && bn.Cmp(ux_ffff_ffff) < 0 {
		s := B(0).Sub(bn, u_65536)
		return B(0).Add(u_65536, fn(s))
	}
	if bn.Cmp(ux_1_0000_0000) > 0 && bn.Cmp(ux_ffff_ffff_ffff_ffff) < 0 {
		return B(0).Or(hi, Fynd(lo, fn))
	}
	return bn
}

// makeAddr converts a patp to address
func makeAddr(name string) *big.Int {
	syls := patp2syls(name)

	addr := B(0)
	for k, v := range syls {
		idx := 0
		if k%2 != 0 || len(syls) == 1 {
			idx = findIndex(suffixes, v)
		} else {
			idx = findIndex(prefixes, v)
		}
		addr.Mul(addr, ux_100).Add(addr, B(int64(idx)))
	}
	return addr
}

func tail(arg *big.Int) *big.Int {
	c := fen(4, u_65535, u_65536, arg)
	if c.Cmp(ux_ffff_ffff) < 0 {
		return c
	}

	return fen(4, u_65535, u_65536, c)
}

func feis(arg *big.Int) *big.Int {
	c := fe(4, u_65535, u_65536, arg)
	if c.Cmp(ux_ffff_ffff) < 0 {
		return c
	}
	return fe(4, u_65535, u_65536, c)
}

func fenLoop(j int, ell *big.Int, arr *big.Int, b *big.Int) *big.Int {
	a := u_65535
	if j < 1 {
		tem1 := B(0).Mul(a, arr)
		return B(0).Add(tem1, ell)
	}
	eff := prf(j-1, ell)

	tmp := B(0)
	if j%2 != 0 {
		ef := B(0).Mod(eff, a)
		tmp = tmp.Add(arr, a).Sub(tmp, ef).Mod(tmp, a)
	} else {
		ef := B(0).Mod(eff, b)
		tmp = tmp.Add(arr, b).Sub(tmp, ef).Mod(tmp, b)
	}

	return fenLoop(j-1, tmp, ell, b)
}

func fen(r int, a *big.Int, b *big.Int, m *big.Int) *big.Int {
	ahh := B(0).Mod(m, a)

	ale := B(0).Div(m, a)

	L := ale
	if ale == a {
		L = ahh
	}

	R := ale
	if ale != a {
		R = ahh
	}

	return fenLoop(r, L, R, b)
}

func feLoop(r int, j int, ell *big.Int, arr *big.Int, b *big.Int) *big.Int {
	a := u_65535
	if j > r {
		if arr.Cmp(a) == 0 {
			tem1 := B(0).Mul(a, arr)
			return B(0).Add(tem1, ell)
		}
		tem1 := B(0).Mul(a, ell)
		return B(0).Add(tem1, arr)
	}
	eff := prf(j-1, arr)

	tmp := B(0)
	if j%2 != 0 {
		tmp = tmp.Add(ell, eff).Mod(tmp, a)
	} else {
		tmp = tmp.Add(ell, eff).Mod(tmp, b)
	}

	return feLoop(r, j+1, arr, tmp, b)
}

func fe(r int, a *big.Int, b *big.Int, m *big.Int) *big.Int {
	L := B(0).Mod(m, a)
	R := B(0).Div(m, a)
	return feLoop(r, 1, L, R, b)
}

func prf(j int, arg *big.Int) *big.Int {
	raku := map[int]uint32{
		0: 0xb76d5eed,
		1: 0xee281300,
		2: 0x85bcae01,
		3: 0x4b387af7,
	}
	return B(int64(Muk(raku[j], 2, arg)))
}

func patp2syls(name string) []string {
	re := regexp.MustCompile(`[~-]`)
	return Chunks(re.ReplaceAllString(name, ""), 3)
}

func findIndex(list [256]string, name string) int {
	for k, v := range list {
		if v == name {
			return k
		}
	}
	return 0
}

func isValidPat(name string) bool {
	if len(name) < 4 {
		return false
	}
	if !strings.HasPrefix(name, "~") {
		return false
	}
	return true
}

// https://github.com/igrmk/golang-chunks-benchmarks/blob/master/chunks.go
func Chunks(s string, chunkSize int) []string {
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string
	chunk := make([]rune, chunkSize)
	len := 0
	for _, r := range s {
		chunk[len] = r
		len++
		if len == chunkSize {
			chunks = append(chunks, string(chunk))
			len = 0
		}
	}
	if len > 0 {
		chunks = append(chunks, string(chunk[:len]))
	}
	return chunks
}
