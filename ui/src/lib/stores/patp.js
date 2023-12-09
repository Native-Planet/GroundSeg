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

function getClass(ship) {
  const hyphens = (ship.match(/-/g) || []).length;
  switch (hyphens) {
    case 0: return ship.length === 3 ? 1 : 2; // Galaxy or Star
    case 1: return 3; // Planet
    case 3: return 4; // Moon
    default: return 5; // Unknown or malformed
  }
}

function tieredAlphabeticalSort(ships) {
  return ships.sort((a, b) => {
    const classA = getClass(a);
    const classB = getClass(b);
    if (classA === classB) {
      return a.localeCompare(b);
    }
    return classA - classB;
  });
}

function tierSplit(ship) {
  const galaxy = ship.slice(-3);
  const starPart = ship.slice(-6, -3);
  const planetPart = ship.slice(-13, -7);
  const moonPart = ship.slice(0, -14);
  return [ moonPart, planetPart, starPart, galaxy ];
}

function tierJoin(parts) {
  const [ moonPart, planetPart, starPart, galaxy ] = parts;
  return `${moonPart}-${planetPart}-${starPart}${galaxy}`.replace(/^-*/, '');
}

function hierarchicalSort(ships) {
  const reversed = ships.map(ship => {
    return tierSplit(ship).reverse().join('.');
  });
  const sorted = reversed.sort()
  return sorted.map(string => tierJoin(string.split('.').reverse()));
}

export const sortModes = {
  alphabetical: tieredAlphabeticalSort,
  hierarchical: hierarchicalSort,
};