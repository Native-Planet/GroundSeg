package co

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	ugi "github.com/deelawn/urbit-gob/internal"
	"github.com/deelawn/urbit-gob/ob"
)

const (
	namePartitionPattern string = ".{1,3}"
	pre                  string = "dozmarbinwansamlitsighidfidlissogdirwacsabwissib" +
		"rigsoldopmodfoglidhopdardorlorhodfolrintogsilmir" +
		"holpaslacrovlivdalsatlibtabhanticpidtorbolfosdot" +
		"losdilforpilramtirwintadbicdifrocwidbisdasmidlop" +
		"rilnardapmolsanlocnovsitnidtipsicropwitnatpanmin" +
		"ritpodmottamtolsavposnapnopsomfinfonbanmorworsip" +
		"ronnorbotwicsocwatdolmagpicdavbidbaltimtasmallig" +
		"sivtagpadsaldivdactansidfabtarmonranniswolmispal" +
		"lasdismaprabtobrollatlonnodnavfignomnibpagsopral" +
		"bilhaddocridmocpacravripfaltodtiltinhapmicfanpat" +
		"taclabmogsimsonpinlomrictapfirhasbosbatpochactid" +
		"havsaplindibhosdabbitbarracparloddosbortochilmac" +
		"tomdigfilfasmithobharmighinradmashalraglagfadtop" +
		"mophabnilnosmilfopfamdatnoldinhatnacrisfotribhoc" +
		"nimlarfitwalrapsarnalmoslandondanladdovrivbacpol" +
		"laptalpitnambonrostonfodponsovnocsorlavmatmipfip"
	suf string = "zodnecbudwessevpersutletfulpensytdurwepserwylsun" +
		"rypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnex" +
		"lunmeplutseppesdelsulpedtemledtulmetwenbynhexfeb" +
		"pyldulhetmevruttylwydtepbesdexsefwycburderneppur" +
		"rysrebdennutsubpetrulsynregtydsupsemwynrecmegnet" +
		"secmulnymtevwebsummutnyxrextebfushepbenmuswyxsym" +
		"selrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpel" +
		"syptermebsetdutdegtexsurfeltudnuxruxrenwytnubmed" +
		"lytdusnebrumtynseglyxpunresredfunrevrefmectedrus" +
		"bexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermer" +
		"tenlusnussyltecmexpubrymtucfyllepdebbermughuttun" +
		"bylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmyl" +
		"wedducfurfexnulluclennerlexrupnedlecrydlydfenwel" +
		"nydhusrelrudneshesfetdesretdunlernyrsebhulryllud" +
		"remlysfynwerrycsugnysnyllyndyndemluxfedsedbecmun" +
		"lyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

	ShipClassEmpty  string = ""
	ShipClassGalaxy string = "galaxy"
	ShipClassStar   string = "star"
	ShipClassPlanet string = "planet"
	ShipClassMoon   string = "moon"
	ShipClassComet  string = "comet"
)

var (

	// Numbers
	zero  = big.NewInt(0)
	one   = big.NewInt(1)
	two   = big.NewInt(2)
	three = big.NewInt(3)
	four  = big.NewInt(4)
	five  = big.NewInt(5)
	eight = big.NewInt(8)

	// Prefixes is the slice of three letter strings that can be used as the first
	// of two syllables in a syllable pair that makes up a ship name.
	Prefixes = regexp.MustCompile(namePartitionPattern).FindAllString(pre, -1)
	// Suffixes is the slice of three letter strings that can be used as the second
	// of two syllables, or one of one syllables in the case of galaxies, that make up a ship name.
	Suffixes = regexp.MustCompile(namePartitionPattern).FindAllString(suf, -1)

	prefixesIndex = map[string]int{}
	suffixesIndex = map[string]int{}
	prefixes      = make([]string, len(Prefixes))
	suffixes      = make([]string, len(Suffixes))
)

func init() {

	// Copy the prefixes and suffixes slices so that any changes made by someone
	// aren't used when doing calculations
	_ = copy(prefixes, Prefixes)
	_ = copy(suffixes, Suffixes)

	// This assumes length of prefixes and suffixes are the same, which they should be.
	for i := 0; i < len(Prefixes); i++ {
		prefixesIndex[Prefixes[i]] = i
		suffixesIndex[Suffixes[i]] = i
	}
}

func patp2syls(name string) []string {

	removeCharsPattern := regexp.MustCompile(`[\^~-]`)
	normalizedName := removeCharsPattern.ReplaceAllString(name, "")
	partitionPattern := regexp.MustCompile(namePartitionPattern)
	return partitionPattern.FindAllString(normalizedName, -1)
}

func bex(n *big.Int) *big.Int {

	return big.NewInt(0).Exp(two, n, nil)
}

func rsh(a, b, c *big.Int) *big.Int {

	return big.NewInt(0).Div(c, bex(big.NewInt(0).Mul(bex(a), b)))
}

func met(a, b, c *big.Int) *big.Int {

	if c == nil {
		c = big.NewInt(0)
	}

	if b.Cmp(zero) == 0 {
		return c
	}

	return met(a, rsh(a, one, b), big.NewInt(0).Add(c, one))
}

func end(a, b, c *big.Int) *big.Int {

	return big.NewInt(0).Mod(c, bex(big.NewInt(0).Mul(bex(a), b)))
}

// Hex2Patp converts a hex-encoded string to a @p-encoded string.
func Hex2Patp(hex string) (string, error) {

	v, ok := big.NewInt(0).SetString(hex, 16)
	if !ok {
		return "", fmt.Errorf(ugi.ErrInvalidHex, hex)
	}

	return Patp(v.String())
}

// Patp2Hex converts a @p-encoded string to a hex-encoded string.
func Patp2Hex(name string) (string, error) {

	if !IsValidPat(name) {
		return "", fmt.Errorf(ugi.ErrInvalidP, name)
	}

	syls := patp2syls(name)

	var addr string
	hasLengthOne := len(syls) == 1
	for i := 0; i < len(syls); i++ {
		if i%2 != 0 || hasLengthOne {
			addr += syl2bin(suffixesIndex[syls[i]])
		} else {
			addr += syl2bin(prefixesIndex[syls[i]])
		}
	}

	bigAddr, ok := big.NewInt(0).SetString(addr, 2)
	if !ok {
		return "", fmt.Errorf(ugi.ErrInvalidBin, addr)
	}

	v, err := ob.Fynd(bigAddr)
	if err != nil {
		return "", nil
	}

	hex := v.Text(16)

	if len(hex)%2 != 0 {
		return "0" + hex, nil
	}

	return hex, nil
}

func syl2bin(idx int) string {

	binStr := strconv.FormatInt(int64(idx), 2)
	return strings.Repeat("0", 8-len(binStr)) + binStr // padStart
}

func patp2bn(name string) (*big.Int, error) {

	hexStr, err := Patp2Hex(name)
	if err != nil {
		return nil, err
	}

	hex, ok := big.NewInt(0).SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf(ugi.ErrInvalidHex, hexStr)
	}

	return hex, nil
}

// Patp2Point converts a @p-encoded string to a big.Int pointer.
func Patp2Point(name string) (*big.Int, error) {
	point, err := patp2bn(name)
	if err != nil {
		return nil, err
	}

	return point, nil
}

// Point2Patp converts a big.Int pointer to a @p-encoded string.
func Point2Patp(point *big.Int) (string, error) {
	return Patp(point.String())
}

// Patp2Dec converts a @p-encoded string to a decimal-encoded string.
func Patp2Dec(name string) (string, error) {

	dec, err := Patp2Point(name)
	if err != nil {
		return "", err
	}

	return dec.String(), nil
}

func patq(arg string) (string, error) {

	v, ok := big.NewInt(0).SetString(arg, 10)
	if !ok {
		return "", fmt.Errorf(ugi.ErrInvalidInt, arg)
	}

	buf := v.Bytes()
	// This is needed for a value of zero
	if len(buf) == 0 {
		buf = []byte{0}
	}
	return buf2patq(buf), nil
}

// Patq converts a string-encoded int or *big.Int to a @q-encoded string.
func Patq(arg interface{}) (string, error) {
	switch v := arg.(type) {
	case string:
		return patq(v)
	case *big.Int:
		return patq(v.String())
	default:
		return "", fmt.Errorf(ugi.ErrInvalidQ, v)
	}
}

// Patq2Point converts a @q-encoded string to a big.Int pointer.
func Patq2Point(name string) (*big.Int, error) {
	point, err := patq2bn(name)
	if err != nil {
		return nil, err
	}

	return point, nil
}

// Point2Patq converts a big.Int pointer to a @q-encoded string.
func Point2Patq(point *big.Int) (string, error) {
	return Patq(point.String())
}

func buf2patq(buf []byte) string {

	var chunked [][]byte
	if len(buf)%2 != 0 && len(buf) > 1 {
		chunked = append([][]byte{{buf[0]}}, chunk(buf[1:], 2)...)
	} else {
		chunked = chunk(buf, 2)
	}

	patq := "~"
	chunkedLen := len(chunked)
	for _, elem := range chunked {

		if patq != "~" {
			patq += "-"
		}

		patq += alg(elem, chunkedLen)
	}

	return patq
}

func prefixName(pair []byte) string {

	if len(pair) == 1 {
		return prefixes[0] + suffixes[pair[0]]
	}

	return prefixes[pair[0]] + suffixes[pair[1]]
}

func name(pair []byte) string {

	if len(pair) == 1 {
		return suffixes[pair[0]]
	}

	return prefixes[pair[0]] + suffixes[pair[1]]
}

func alg(pair []byte, chunkedLen int) string {

	if len(pair)%2 != 0 && chunkedLen > 1 {
		return prefixName(pair)
	}

	return name(pair)
}

func chunk(items []byte, size int) [][]byte {

	slices := [][]byte{}

	for _, item := range items {

		sliceLength := len(slices)
		if sliceLength == 0 || len(slices[sliceLength-1]) == size {
			slices = append(slices, []byte{})
			sliceLength++
		}

		slices[sliceLength-1] = append(slices[sliceLength-1], item)
	}

	return slices
}

// Hex2Patq converts a hex-encoded string to a @q-encoded string.
// Note that this preserves leading zero bytes.
func Hex2Patq(arg string) (string, error) {

	hexStr := arg
	if len(arg)%2 != 0 {
		hexStr = "0" + hexStr
	}

	buf, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf(ugi.ErrInvalidHex, arg)
	}

	return buf2patq(buf), nil
}

// Patq2Hex converts a @q-encoded string to a hex-encoded string.
// Note that this preserves leading zero bytes.
func Patq2Hex(name string) (string, error) {

	if !IsValidPat(name) {
		return "", fmt.Errorf(ugi.ErrInvalidQ, name)
	}

	if len(name) == 0 {
		return "00", nil
	}

	chunks := strings.Split(name[1:], "-")
	return splat(chunks), nil
}

func dec2hex(dec int) string {

	decStr := strconv.FormatInt(int64(dec), 16)
	if l := len(decStr); l < 2 {
		padding := strings.Repeat("0", 2-l)
		decStr = padding + decStr
	}

	return decStr
}

func splat(chunks []string) string {

	var hexStr string
	for _, chunk := range chunks {

		syls := []string{chunk}
		if len(chunk) > 3 {
			syls = []string{chunk[:3], chunk[3:]}
		}
		if len(syls) == 1 {
			hexStr += dec2hex(suffixesIndex[syls[0]])
		} else {
			hexStr += dec2hex(prefixesIndex[syls[0]]) + dec2hex(suffixesIndex[syls[1]])
		}
	}

	return hexStr
}

func patq2bn(name string) (*big.Int, error) {

	hexStr, err := Patq2Hex(name)
	if err != nil {
		return nil, err
	}

	v, ok := big.NewInt(0).SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf(ugi.ErrInvalidHex, name)
	}

	return v, nil
}

// Patq2Dec converts a @q-encoded string to a decimal-encoded string.
func Patq2Dec(name string) (string, error) {

	v, err := patq2bn(name)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

// Clan determines the ship class of a @p value.
func Clan(who string) (string, error) {

	name, err := patp2bn(who)
	if err != nil {
		return ShipClassEmpty, err
	}

	wid := met(three, name, nil)

	if wid.Cmp(one) <= 0 {
		return ShipClassGalaxy, nil
	}
	if wid.Cmp(two) <= 0 {
		return ShipClassStar, nil
	}
	if wid.Cmp(four) <= 0 {
		return ShipClassPlanet, nil
	}
	if wid.Cmp(eight) <= 0 {
		return ShipClassMoon, nil
	}

	return ShipClassComet, nil
}

// ClanPoint determines the ship class of a big.Int-encoded @p value.
func ClanPoint(arg *big.Int) (string, error) {
	patp, err := Patp(arg)
	if err != nil {
		return "", err
	}
	return Clan(patp)
}

// Sein determines the parent of a @p value.
func Sein(name string) (string, error) {

	who, err := patp2bn(name)
	if err != nil {
		return "", err
	}

	mir, err := Clan(name)
	if err != nil {
		return "", err
	}

	var res *big.Int
	switch mir {
	case ShipClassGalaxy:
		res = who
	case ShipClassStar:
		res = end(three, one, who)
	case ShipClassPlanet:
		res = end(four, one, who)
	case ShipClassMoon:
		res = end(five, one, who)
	default:
		res = zero
	}

	return Patp(res.String())
}

// SeinPoint determines the parent of a big.Int-encoded @p value.
func SeinPoint(arg *big.Int) (*big.Int, error) {
	patp, err := Point2Patp(arg)
	if err != nil {
		return nil, err
	}
	sein, err := Sein(patp)
	if err != nil {
		return nil, err
	}

	return Patp2Point(sein)
}

/*
IsValidPat weakly checks if a string is a valid @p or @q value.

This is, at present, a pretty weak sanity check.  It doesn't confirm the
structure precisely (e.g. dashes), and for @q, it's required that q values
of (greater than one) odd bytelength have been zero-padded.  So, for
example, '~doznec-binwod' will be considered a valid @q, but '~nec-binwod'
will not.
*/
func IsValidPat(name string) bool {

	if len(name) < 4 || name[0] != '~' {
		return false
	}

	syls := patp2syls(name)

	sylsLen := len(syls)
	for i, syl := range syls {
		if i%2 != 0 || sylsLen == 1 {
			if _, ok := suffixesIndex[syl]; !ok {
				return false
			}
		} else if _, ok := prefixesIndex[syl]; !ok {
			return false
		}
	}

	return !(sylsLen%2 != 0 && sylsLen != 1)
}

// IsValidPatp validates a @p string.
func IsValidPatp(str string) bool {

	dec, err := Patp2Dec(str)
	if err != nil {
		return false
	}

	p, err := Patp(dec)
	if err != nil {
		return false
	}

	return IsValidPat(str) && str == p
}

// IsValidPatq validates a @q string.
func IsValidPatq(str string) bool {

	dec, err := Patq2Dec(str)
	if err != nil {
		return false
	}

	q, err := Patq(dec)
	if err != nil {
		return false
	}

	isValid, err := EqPatq(str, q)
	if err != nil {
		return false
	}

	return IsValidPat(str) && isValid
}

func removeLeadingZeros(str string) string {

	for i, c := range str {
		if c != '0' {
			return str[i:]
		}
	}

	return ""
}

func eqModLeadingZeros(s, t string) bool {

	return removeLeadingZeros(s) == removeLeadingZeros(t)
}

// EqPatq performs an equality comparison on @q values.
func EqPatq(p, q string) (bool, error) {

	phex, err := Patq2Hex(p)
	if err != nil {
		return false, err
	}

	qhex, err := Patq2Hex(q)
	if err != nil {
		return false, err
	}

	return eqModLeadingZeros(phex, qhex), nil
}

func patp(arg string) (string, error) {
	v, ok := big.NewInt(0).SetString(arg, 10)
	if !ok {
		return "", fmt.Errorf(ugi.ErrInvalidInt, arg)
	}

	sxz, err := ob.Fein(v.String())
	if err != nil {
		return "", err
	}

	dyy := met(four, sxz, nil)
	dyx := met(three, sxz, nil)

	p := "~"

	if dyx.Cmp(one) <= 0 {
		p += suffixes[int(sxz.Int64())]
	} else {
		p += patpLoop(dyy, sxz, zero, "")
	}

	return p, nil
}

// Patp converts a either a string-encoded int or *big.Int to a @p-encoded string.
func Patp(arg interface{}) (string, error) {
	switch v := arg.(type) {
	case string:
		return patp(v)
	case *big.Int:
		return patp(v.String())
	default:
		return "", fmt.Errorf(ugi.ErrInvalidP, arg)
	}
}

func patpLoop(dyy, tsxz, timp *big.Int, trep string) string {

	log := end(four, one, tsxz)
	pre := prefixes[int(rsh(three, one, log).Int64())]
	suf := suffixes[int(end(three, one, log).Int64())]

	var etc string
	if big.NewInt(0).Mod(timp, four).Cmp(zero) == 0 {
		if timp.Cmp(zero) != 0 {
			etc = "--"
		}
	} else {
		etc = "-"
	}

	res := pre + suf + etc + trep

	if timp.Cmp(dyy) == 0 {
		return trep
	}

	return patpLoop(dyy, rsh(four, one, tsxz), big.NewInt(0).Add(timp, one), res)
}
