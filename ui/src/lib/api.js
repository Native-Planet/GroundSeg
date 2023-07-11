import { writable } from 'svelte/store'
export const webuiVersion = 'v1.4.2'

//
// fade transition params
//
export const fadeIn = {duration:200, delay: 160}
export const fadeOut = {duration:200}

//
// writable stores
//
export const noconn = writable(false)
export const linuxUpdate = writable(false)
export const startram = writable({anchor: {wgReg:false, wgRunning: false}})
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
  updateAnchor(update['anchor'])
  updateLinux(update['system_update'])
  updateConnStatus(update['status'])
	updateUrbits(update['urbits'])
  updateSystemInformation(update['system'])
}

const updateAnchor = a => {if (a) {startram.set(a)}}
const updateLinux = u => {if (u) {linuxUpdate.set(u)}}
const updateUrbits = p => {if (p) {urbits.set(p)}}
const updateSystemInformation = s => {if (s) {system.set(s)}}

let noconncount = 0
const updateConnStatus = c => {
  if (c == 'noconn') {
    if (noconncount < 3) {
      ++noconncount
      console.log("No connection to GroundSeg API (" + noconncount + ")")
    } else {
      noconn.set(true)
    }
  } else {
    noconn.set(false)
    noconncount = 0
  }
}

//
// misc
//

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
