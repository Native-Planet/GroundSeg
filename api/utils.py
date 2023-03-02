# Python
import ssl
import hashlib
import urllib.request
from time import sleep

# Modules
import nmcli

# GroundSeg modules
from log import Log

class Utils:
    def make_hash(file):
        h  = hashlib.sha256()
        b  = bytearray(128*1024)
        mv = memoryview(b)
        with open(file, 'rb', buffering=0) as f:
            while n := f.readinto(mv):
                h.update(mv[:n])
        return h.hexdigest()

    def check_patp(patp):
        # Make sure patp is string
        if type(patp) != str:
            return False

        # Remove sig from patp
        if patp.startswith("~"):
            patp = patp[1:]

        # valid
        pre = "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
        suf = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

        # convert to array
        pre = [pre[i:i+3] for i in range(0, len(pre), 3)]
        suf = [suf[i:i+3] for i in range(0, len(suf), 3)]

        # Galaxy check
        if len(patp) == 3:
            return patp in suf

        # Split patp
        patp = patp.split("-")

        # Check if valid
        for p in patp:
            if len(p) == 6:
                if p[:3] not in pre:
                    return False
                if p[3:] not in suf:
                    return False
            else:
                return False

        return True

    def check_internet_access():
        Log.log("Updater: Checking internet access")
        try:
            context = ssl._create_unverified_context()
            urllib.request.urlopen('https://nativeplanet.io',
                                   timeout=1,
                                   context=context)

            return True

        except Exception as e:
            Log.log(f"Updater: Check internet access error: {e}")
            return False

    def get_wifi_device():
        for d in nmcli.device():
            if d.device_type == 'wifi':
                return d.device
        return "none"

    def list_wifi_ssids():
        return [x.ssid for x in nmcli.device.wifi() if len(x.ssid) > 0]

    def wifi_connect(ssid, pwd):
        try:
            nmcli.device.wifi_connect(ssid, pwd)
            Log.log(f"WiFi: Connected to: {ssid}")
            return True
        except Exception as e:
            Log.log(f"WiFi: Failed to connect to network: {e}")
            return False

    def start_script():
        return """\
#!/bin/bash

set -eu
# set defaults
amesPort="34343"
httpPort="80"
loom="31"

# check args
for i in "$@"
do
case $i in
  -p=*|--port=*)
      amesPort="${i#*=}"
      shift
      ;;
   --http-port=*)
      httpPort="${i#*=}"
      shift
      ;;
   --loom=*)
      loom="${i#*=}"
      shift
      ;;
esac
done

# If the container is not started with the `-i` flag
# then STDIN will be closed and we need to start
# Urbit/vere with the `-t` flag.
ttyflag=""
if [ ! -t 0 ]; then
echo "Running with no STDIN"
ttyflag="-t"
fi

# Check if there is a keyfile, if so boot a ship with its name, and then remove the key
if [ -e *.key ]; then
# Get the name of the key
keynames="*.key"
keys=( $keynames )
keyname=''${keys[0]}
mv $keyname /tmp

# Boot urbit with the key, exit when done booting
vere $ttyflag -w $(basename $keyname .key) -k /tmp/$keyname -c $(basename $keyname .key) -p $amesPort -x --http-port $httpPort --loom $loom

# Remove the keyfile for security
rm /tmp/$keyname
rm *.key || true
elif [ -e *.comet ]; then
cometnames="*.comet"
comets=( $cometnames )
cometname=''${comets[0]}
rm *.comet

vere $ttyflag -c $(basename $cometname .comet) -p $amesPort -x --http-port $httpPort --loom $loom
fi

# Find the first directory and start urbit with the ship therein
dirnames="*/"
dirs=( $dirnames )
dirname=''${dirnames[0]}

exec vere $ttyflag -p $amesPort --http-port $httpPort --loom $loom $dirname 
"""
