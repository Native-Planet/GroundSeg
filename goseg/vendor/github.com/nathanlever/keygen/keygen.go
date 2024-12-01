package keygen

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkova/argon2"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"lukechampine.com/urbit/ob"
)

type Generator struct {
	Name, Version string
}

type Meta struct {
	Generator  Generator
	Spec       string
	Ship       uint32
	Patp       string
	Tier       string
	Passphrase string
}

type Keys struct {
	Public, Private, Chain, Address string
}

type Node struct {
	Type           string
	Seed           string
	Keys           Keys
	DerivationPath string
}

type NetworkKeys struct {
	Crypt, Auth KeyPair
}

type KeyPair struct {
	Private, Public string
}

type Network struct {
	Type string
	Seed string
	Keys NetworkKeys
}

type BitcoinKeys struct {
	Xpub, Xprv string
}

type Bitcoin struct {
	Type           string
	Seed           string
	Keys           BitcoinKeys
	DerivationPath string
}

type Wallet struct {
	Meta           Meta
	Ticket         string
	Shards         []string
	Ownership      Node
	Transfer       Node
	Spawn          *Node
	Voting         *Node
	Management     Node
	Network        *Network
	BitcoinTestnet Bitcoin
	BitcoinMainnet Bitcoin
}

func Tier(p ob.AzimuthPoint) string {
	if p.IsGalaxy() {
		return "galaxy"
	} else if p.IsStar() {
		return "star"
	} else {
		return "planet"
	}
}

func patqToByteSlice(patq string) []byte {
	patq2 := strings.TrimPrefix(patq, "~")
	patq2 = strings.ReplaceAll(patq2, "-", "")
	var result []byte
	for i := 0; i < len(patq2); i += 3 {
		syl := patq2[i : i+3]
		result = append(result, phonemeIndex[syl])
	}
	return result
}

func byteSliceToPatq(s []byte) string {
	result := "~"
	if len(s) == 1 {
		return "~" + suffixes[s[0]]
	}
	for i := 0; i < len(s); i += 2 {
		result += prefixes[s[i]]
		result += suffixes[s[i+1]]
		if i+2 != len(s) {
			result += "-"
		}
	}
	return result
}

func shard(ticket string) []string {
	ticketSlice := patqToByteSlice(ticket)

	if len(ticketSlice) != 48 {
		return []string{ticket}
	}

	s0 := ticketSlice[:32]
	s1 := ticketSlice[16:]
	ticketSliceCopy := make([]byte, 48)
	copy(ticketSliceCopy, ticketSlice)
	s2 := append(ticketSliceCopy[:16], ticketSliceCopy[32:]...)

	shards := []string{
		byteSliceToPatq(s0),
		byteSliceToPatq(s1),
		byteSliceToPatq(s2),
	}

	combined1, _ := Combine(&shards[0], &shards[1], nil)
	combined2, _ := Combine(&shards[0], nil, &shards[2])
	combined3, _ := Combine(nil, &shards[1], &shards[2])

	if !(combined1 == ticket &&
		combined2 == ticket &&
		combined3 == ticket) {
		panic("produced invalid shards -- please report this as a bug")
	}

	return shards
}

func Combine(s0, s1, s2 *string) (string, error) {
	if s0 != nil && s1 != nil {
		lastThird := *s1
		return *s0 + lastThird[56:], nil
	}
	if s0 != nil && s2 != nil {
		lastThird := *s2
		return *s0 + lastThird[56:], nil
	}
	if s1 != nil && s2 != nil {
		firstThird := *s2
		s1NoPrefix := *s1
		return firstThird[:57] + s1NoPrefix[1:], nil
	}
	return "", errors.New("combine: need at least two shards")
}

func deriveNodeSeed(master []byte, walletType string) string {
	data := append(master, walletType...)
	hash := sha256.Sum256(data)
	result, err := bip39.NewMnemonic(hash[:])
	if err != nil {
		panic("invalid entropy in deriveNodeSeed")
	}
	return result
}

func deriveNodeKeys(mnemonic, derivationPath, passphrase string) Keys {
	seed := bip39.NewSeed(mnemonic, passphrase)
	hd, err := bip32.NewMasterKey(seed)
	if err != nil {
		panic("invalid masterKey in deriveNodeKeys")
	}
	wallet := derivePath(hd, derivationPath)
	publicKey := fmt.Sprintf("%x", wallet.PublicKey().Key)
	privateKey := fmt.Sprintf("%x", wallet.Key)
	chainCode := fmt.Sprintf("%x", wallet.ChainCode)
	address := addressFromSecp256k1Public(wallet.PublicKey().Key)
	return Keys{
		Public:  publicKey,
		Private: privateKey,
		Chain:   chainCode,
		Address: address,
	}
}

func derivePath(hd *bip32.Key, derivationPath string) *bip32.Key {
	dp := strings.TrimPrefix(derivationPath, "m/")
	dpSlice := strings.Split(dp, "/")
	childKey := hd
	for _, v := range dpSlice {
		value, err := strconv.ParseUint(strings.TrimSuffix(v, "'"), 10, 32)
		if err != nil {
			panic("invalid derivationPath")
		}
		if strings.HasSuffix(v, "'") {
			value = value + 2147483648
		}
		childKey, err = childKey.NewChildKey(uint32(value))
		if err != nil {
			panic("invalid childKey in derivePath")
		}
	}
	return childKey
}

func addressFromSecp256k1Public(pub []byte) string {
	p, err := crypto.DecompressPubkey(pub)
	if err != nil {
		panic("invalid public key")
	}
	address := crypto.PubkeyToAddress(*p).String()
	return address
}

func deriveNode(master []byte, walletType, derivationPath, passphrase string) Node {
	mnemonic := deriveNodeSeed(master, walletType)
	keys := deriveNodeKeys(mnemonic, derivationPath, passphrase)
	return Node{
		Type:           walletType,
		Seed:           mnemonic,
		Keys:           keys,
		DerivationPath: derivationPath,
	}
}

func deriveNetworkSeed(mnemonic string, revision uint, passphrase string) [32]byte {
	seed := bip39.NewSeed(mnemonic, passphrase)
	data := append(seed, "network"...)
	data = append(data, fmt.Sprint(revision)...)
	hash := sha256.Sum256(data)
	if revision != 0 {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

func reverse(s []byte) []byte {
	result := make([]byte, len(s))
	for i, v := range s {
		result[len(s)-i-1] = v
	}
	return result
}

func deriveNetworkKeys(seed [32]byte) NetworkKeys {
	hash := sha512.Sum512(reverse(seed[:]))

	c := hash[32:]
	a := hash[:32]

	crypt := ed25519.NewKeyFromSeed(c)
	cpub, ok := crypt.Public().(ed25519.PublicKey)
	if !ok {
		panic("could not assert the public key to ed25519 public key")
	}
	auth := ed25519.NewKeyFromSeed(a)
	apub, ok := auth.Public().(ed25519.PublicKey)
	if !ok {
		panic("could not assert the public key to ed25519 public key")
	}

	cryptKeyPair := KeyPair{
		Private: fmt.Sprintf("%x", reverse(c)),
		Public:  fmt.Sprintf("%x", reverse([]byte(cpub))),
	}
	authKeyPair := KeyPair{
		Private: fmt.Sprintf("%x", reverse(a)),
		Public:  fmt.Sprintf("%x", reverse([]byte(apub))),
	}

	return NetworkKeys{
		Crypt: cryptKeyPair,
		Auth:  authKeyPair,
	}
}

func deriveNetworkInfo(mnemonic string, revision uint, passphrase string) Network {
	seed := deriveNetworkSeed(mnemonic, revision, passphrase)
	keys := deriveNetworkKeys(seed)
	return Network{
		Type: "network",
		Seed: fmt.Sprintf("%x", seed),
		Keys: keys,
	}
}

func deriveBitcoin(
	master []byte,
	walletType,
	derivationPath,
	passphrase string,
	pubVersion,
	prvVersion []byte,
) Bitcoin {
	mnemonic := deriveNodeSeed(master, walletType)
	keys := deriveBitcoinKeys(mnemonic, derivationPath, passphrase, pubVersion, prvVersion)
	return Bitcoin{
		Type:           walletType,
		Seed:           mnemonic,
		Keys:           keys,
		DerivationPath: derivationPath,
	}
}

func deriveBitcoinKeys(
	mnemonic,
	derivationPath,
	passphrase string,
	pubVersion,
	prvVersion []byte,
) BitcoinKeys {
	seed := bip39.NewSeed(mnemonic, passphrase)
	hd, err := bip32.NewMasterKey(seed)
	if err != nil {
		panic("invalid masterKey in deriveBitcoinKeys")
	}
	wallet := derivePath(hd, derivationPath)

	public := wallet.PublicKey()
	public.Version = pubVersion
	public.ChildNumber = []byte{0x00, 0x00, 0x00, 0x00}
	public.FingerPrint = []byte{0x00, 0x00, 0x00, 0x00}
	public.Depth = 0x00
	xpub := public.String()

	wallet.Version = prvVersion
	wallet.ChildNumber = []byte{0x00, 0x00, 0x00, 0x00}
	wallet.FingerPrint = []byte{0x00, 0x00, 0x00, 0x00}
	wallet.Depth = 0x00
	xprv := wallet.String()

	return BitcoinKeys{
		Xpub: xpub,
		Xprv: xprv,
	}
}

func (w *Wallet) UnmarshalJSON(b []byte) error {
	type wallet2 Wallet
	var w2 wallet2
	err := json.Unmarshal(b, &w2)
	if err != nil {
		return err
	}

	*w = Wallet(w2)

	if w.Spawn.Type == "" {
		w.Spawn = nil
	}
	if w.Voting.Type == "" {
		w.Voting = nil
	}
	if w.Network.Type == "" {
		w.Network = nil
	}

	return nil
}

func GenerateWallet(
	ticket string,
	ship uint32,
	passphrase string,
	revision uint,
	boot bool,
) Wallet {
	generator := Generator{
		Name:    "go-key-generation",
		Version: "0.0.1",
	}
	metaShip := ob.AzimuthPoint(ship)
	patp := metaShip.String()
	tier := Tier(metaShip)
	meta := Meta{
		Generator:  generator,
		Spec:       "UP8",
		Ship:       ship,
		Patp:       patp,
		Tier:       tier,
		Passphrase: passphrase,
	}
	ticketByteSlice := patqToByteSlice(ticket)
	urbitKeyGen := "urbitkeygen" + fmt.Sprint(ship)
	masterSeed := argon2.UKey(
		ticketByteSlice,
		[]byte(urbitKeyGen),
		1,
		512000,
		4,
		32,
	)
	ownership := deriveNode(
		masterSeed,
		"ownership",
		"m/44'/60'/0'/0/0",
		passphrase,
	)
	transfer := deriveNode(
		masterSeed,
		"transfer",
		"m/44'/60'/0'/0/0",
		passphrase,
	)
	var spawn *Node
	if !metaShip.IsPlanet() {
		s := deriveNode(
			masterSeed,
			"spawn",
			"m/44'/60'/0'/0/0",
			passphrase,
		)
		spawn = &s
	}
	var voting *Node
	if metaShip.IsGalaxy() {
		v := deriveNode(
			masterSeed,
			"voting",
			"m/44'/60'/0'/0/0",
			passphrase,
		)
		voting = &v
	}
	management := deriveNode(
		masterSeed,
		"management",
		"m/44'/60'/0'/0/0",
		passphrase,
	)
	var network *Network
	if boot {
		n := deriveNetworkInfo(
			management.Seed,
			revision,
			passphrase,
		)
		network = &n
	}

	bitcoinTestnet := deriveBitcoin(
		masterSeed,
		"bitcoinTestnet",
		"m/84'/1'/0'",
		passphrase,
		[]byte{0x04, 0x5f, 0x1c, 0xf6},
		[]byte{0x04, 0x5f, 0x18, 0xbc},
	)

	bitcoinMainnet := deriveBitcoin(
		masterSeed,
		"bitcoinMainnet",
		"m/84'/0'/0'",
		passphrase,
		[]byte{0x04, 0xb2, 0x47, 0x46},
		[]byte{0x04, 0xb2, 0x43, 0x0c},
	)

	return Wallet{
		Meta:           meta,
		Ticket:         ticket,
		Shards:         shard(ticket),
		Ownership:      ownership,
		Transfer:       transfer,
		Spawn:          spawn,
		Voting:         voting,
		Management:     management,
		Network:        network,
		BitcoinTestnet: bitcoinTestnet,
		BitcoinMainnet: bitcoinMainnet,
	}
}

var phonemeIndex = func() map[string]uint8 {
	m := make(map[string]uint8)
	for i, p := range prefixes {
		m[p] = uint8(i)
	}
	for i, s := range suffixes {
		m[s] = uint8(i)
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
