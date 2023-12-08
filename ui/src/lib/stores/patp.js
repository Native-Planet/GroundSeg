// removes ~ from patp
export const sigRemove = patp => {
  if (patp != undefined) {
    if (patp.startsWith("~")) {
      patp = patp.substring(1);
    }
  }
  return patp
}

// checks if patp is correct -- use sigRemove() first!
export const checkPatp = patp => {
  if (patp == undefined) {
    return false
  }
  // prefixes and suffixes into arrays
  // Split the string by hyphen
  const wordlist = patp.split("-");
  // Define the regular expression pattern
  const pattern = /(^[a-z]{6}$|^[a-z]{3}$)/;
  // Define pre and suf (truncated for brevity)
  let pre = "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
  let suf = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

  for (const word of wordlist) {
    // Check regular expression match
    if (!pattern.test(word)) {
      return false;
    }

    // Check prefixes and suffixes
    if (word.length > 3) {
      if (!pre.includes(word.substring(0, 3)) || !suf.includes(word.substring(3, 6))) {
        return false;
      }
    } else {
      if (!suf.includes(word)) {
        return false;
      }
    }
  }
  return true;
}