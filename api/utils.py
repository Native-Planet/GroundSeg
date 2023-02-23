# Python
import ssl
import hashlib
import requests
import urllib.request
from time import sleep

# Modules
import nmcli

# GroundSeg modules
from log import Log
from binary_updater import BinUpdater

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
        print(f"patp ({patp})")

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

    def get_version_info(config, debug_mode):
        Log.log("Updater: Thread started")
        while True:
            try:
                Log.log("Updater: Checking for updates")
                url = config.config['updateUrl']
                r = requests.get(url)

                if r.status_code == 200:
                    config.update_avail = True
                    config.update_payload = r.json()

                    # Run binary updater
                    b = BinUpdater()
                    b.check_bin_update(config, debug_mode)

                    if config.gs_ready:
                        print("Updater: placeholder -- docker update here")
                        # Run docker updates
                        sleep(config.config['updateInterval'])
                    else:
                        sleep(60)

                else:
                    raise ValueError(f"Status code {r.status_code}")

            except Exception as e:
                config.update_avail = False
                Log.log(f"Updater: Unable to retrieve update information: {e}")
                sleep(60)

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

    '''
    def remove_urbit_containers():
        client = docker.from_env()

        # Force remove containers
        containers = client.containers.list(all=True)
        for container in containers:
            try:
                if container.image.tags[0] == "tloncorp/urbit:latest":
                    container.remove(force=True)
                if container.image.tags[0] == "tloncorp/vere:latest":
                    container.remove(force=True)
            except:
                pass

        # Check if all have been removed
        containers = client.containers.list(all=True)
        count = 0
        for container in containers:
            try:
                if container.image.tags[0] == "tloncorp/urbit:latest":
                    count = count + 1
                if container.image.tags[0] == "tloncorp/vere:latest":
                    count = count + 1
            except:
                pass
        return count == 0


'''
