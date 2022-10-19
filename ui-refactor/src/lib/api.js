import { writable } from 'svelte/store'

//
// fade transition params
//
export const fadeIn = {duration:200, delay: 160}
export const fadeOut = {duration:200}

//
// writable stores
//
export const piers = writable([])
export const system = writable({})
export const api = writable('')

//
// state update main functions
//
export const updateState = update => {
	updatePiers(update['piers'])
	updateApi(update['api'])
}

export const updateApi = a => {if(a){api.set(a)}}

export const updatePiers = p => {
	//piers.update(state => {
	//})
	if (p) {
	piers.set(p)	
	}
}
/*
export const updateSystem = update => {
	if (update['system']) {
		state.update( s => {
			s['system'] = update['system']
			return s
	})}
}
*/


//
// misc
//
const mergeByProperty = (target, source, prop) => {
  source.forEach(sourceElement => {
    let targetElement = target.find(targetElement => {
      return sourceElement[prop] === targetElement[prop];
    })
    targetElement ? Object.assign(targetElement, sourceElement) : target.push(sourceElement);
  })
}

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
