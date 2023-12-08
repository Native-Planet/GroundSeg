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

/** Pad patp to moon length with 0 and - */
function padPatp(patp) {
  while (patp.length < 27) {
    patp = '0' + patp;
  }
  return patp;
}

/** Remove patp padding */
function unpadPatp(patp) {
  return patp.replace(/^0+/, '');
}

/** Reverses patp, in chunks of 3 */
function reversePatp(patp) {
  const chunks = patp.match(/.{1,3}/g) || []; // Split into chunks of 3 characters
  return chunks.reverse().join('-'); // Reverse the chunks and join with hyphens
}

/** Sort alphabetical but put higher tiers first (IE planets above moons) */
function tieredAlphabeticalSort(ships) {
  return ships.map(padPatp).sort().map(unpadPatp);
}

/** Sort hierarchically so moons are immediately below their planets, etc */
function hierarchicalSort(ships) {
  return ships
    .map(ship => reversePatp(padPatp(ship))) // Pad and then reverse
    .sort()
    .map(ship => unpadPatp(reversePatp(ship))); // Reverse back and then unpad
}

export const sortModes = {
  alphabetical: tieredAlphabeticalSort,
  hierarchical: hierarchicalSort,
}