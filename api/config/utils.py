# Python
import math
import base64
import socket
import psutil
import hashlib
import subprocess
from time import sleep

# Modules
import nmcli
from cryptography.hazmat.primitives.asymmetric.padding import PKCS1v15
from cryptography.hazmat.primitives import serialization

# GroundSeg modules
import config.start_script
import config.pack_script
import config.meld_script

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

        # patps cannot start with doz
        if patp.startswith("doz"):
            return False

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

    def convert_region_data(data):
        try:
            regions = []
            for r in data.keys():
                region = {
                        "name": r,
                        "country": data[r]['country'],
                        "desc": data[r]['desc']
                        }

                regions.append(region)
        except Exception as e:
            print(f"Util: Failed to convert region data: {e}")
            regions = None

        return(regions)


    def check_internet_access(addr):
        print("Updater: Checking internet access")
        try:
            socket.setdefaulttimeout(3)
            host, port = addr.split(":")
            socket.socket(socket.AF_INET, socket.SOCK_STREAM).connect((host, int(port)))
            print("config.utils:check_internet_access Success!")
            return True
        except Exception as e:
            print(f"Updater: Check internet access error: {e}")
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
            print(f"WiFi: Connected to: {ssid}")
            return True
        except Exception as e:
            print(f"WiFi: Failed to connect to network: {e}")
            return False

    def convert_pub(pub):
        converted = ""
        try:
            if pub != "":
                converted = pub.public_bytes(
                    encoding=serialization.Encoding.PEM,
                    format=serialization.PublicFormat.SubjectPublicKeyInfo
                    ).decode("utf-8")
        except Exception as e:
            print(f"Keygen: Failed to convert pubkey: {e}")

        return converted

    def decrypt_password(priv, pwd):
        decrypted = ""
        try:
            pwd_bstr = bytes(pwd,'utf-8')
            pwd_bytes = base64.b64decode(pwd_bstr)
            decrypted = priv.decrypt(pwd_bytes, PKCS1v15()).decode("utf-8")
        except Exception as e:
            print(f"Keygen: Failed to decrypt password: {e}")

        return decrypted

    def compare_password(salt, password, pwHash):
        res = False
        try:
            encoded_str = (salt + password).encode('utf-8')
            this_hash = hashlib.sha512(encoded_str).hexdigest()
            res = this_hash == pwHash
        except Exception as e:
            print(f"Login: Failed to compare passwords: {e}")

        return res

    def start_swap(loc):
        try:
            subprocess.call(["swapon", loc])
        except Exception as e:
            print(f"Swap: Failed to run swapon: {e}")
            return False
        return True

    def stop_swap(loc):
        try:
            subprocess.call(["swapoff", loc])
        except Exception as e:
            print(f"Swap: Failed to run swapoff: {e}")
            return False
        return True

    def make_swap(loc, val):
        try:
            subprocess.call(["fallocate", "-l", f"{val}G", loc])
            subprocess.call(["chmod", "600", loc])
            subprocess.call(["mkswap", loc])
        except Exception as e:
            print(f"Swap: Failed to make swap: {e}")
            return False
        return True

    def active_swap(loc):
        count = 0
        while count < 3:
            try:
                res = subprocess.run(["swapon", "--show"], capture_output=True)
                swap_arr = [x for x in res.stdout.decode("utf-8").split('\n') if loc in x]
                return int("".join(filter(str.isdigit, [x for x in swap_arr[0].split(" ") if x != ""][2])))
            except Exception as e:
                print(f"Swap: Failed to get active swap: {e}")
                count += 1
                sleep(count * 2)

            # Returns None if failed

    def max_swap(loc, val):
        cap = 32 # arbitrary cap for the webui
        free = cap
        try:
            free = math.ceil(psutil.disk_usage(loc).free / (1024 ** 3)) - 2
            if free > cap:
                free = cap
        except Exception as e:
            if val > 0:
                print(f"Swap: Failed to get maximum swap: {e}")
        return free

    '''
    def linux_update_script():
        return """\
#!/bin/bash

# Update package index
sudo apt-get update

# Upgrade packages
sudo apt-get -y upgrade

# Check if a reboot is required
if [ -f /var/run/reboot-required ]; then
  echo "System restart required. Restarting now..."
  sudo reboot
else
  echo "No restart required."
fi"""
    '''

    def pack_script():
        return pack_script.pack_script()

    def meld_script():
        return meld_script.meld_script()

    def start_script():
        return start_script.start_script()
