import { writable } from 'svelte/store'

export const dev = false
export const webuiVersion = 'Beta-3.2.2'

//
// fade transition params
//
export const fadeIn = {duration:200, delay: 160}
export const fadeOut = {duration:200}

//
// writable stores
//
export const secret = writable('')
export const codes = writable({})
export const urbits = writable([])
export const system = writable({})
export const api = writable('')
export const isPortrait = writable(false)
export const currentLog = writable({'container': '', 'log': []})
export const power = writable('')

//
// state update
//
export const updateState = update => {
	updateUrbits(update['urbits'])
  updateSystemInformation(update['system'])
	updateApi(update['api'])
}

const updateApi = a => {if(a){api.set(a)}}
const updateUrbits = p => {if (p) {urbits.set(p)}}
const updateSystemInformation = s => {if (s) {system.set(s)}}

//
// misc
//
// Todo: clean this up

export const isPatp = p => {
  
  // prefixes and suffixes into arrays
  let pre = "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
  let suf = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"
  pre = pre.match(/.{1,3}/g)
  suf = suf.match(/.{1,3}/g)

  // patp into array
  p = p.replace(/~/g,'').split('-')

  // check every syllable
  let checked = []
  for (let i = 0; i < p.length; i++) {

    if (p[i].length == 3) {
      checked.push(suf.includes(p[i]))
    } else if (p[i].length == 6) {
      let s = p[i].match(/.{1,3}/g)
      checked.push(pre.includes(s[0]) && (suf.includes(s[1])))
    } else {return false}
  }

  // returns true if no falses in checked
  return !checked.includes(false)
}
