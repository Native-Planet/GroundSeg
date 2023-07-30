import os
import json
import socket
import string
import base64
import secrets
import hashlib
import platform

from pywgkey.key import WgKey
from crontab import CronTab

from config.swap import Swap

class Config:
    # GroundSeg version
    version = "v2.0.0"

    # Class is ready
    ready = False

    # Version Server
    version_server_ready = False
    version_info = {}

    # System Monitor
    _ram = None
    _cpu = None
    _core_temp = None
    _disk = None

    # WiFi Network Information
    _wifi_enabled = False
    _active_network = None
    _wifi_networks = []

    # Http Upload open
    http_open = False

    # Uploader secret
    upload_secret = ''

    # default content of system.json
    default_system_config = {
            # The setup stages are
            # start -> profile -> startram -> complete
            "setup": "start",
            "endpointUrl": "api.startram.io",
            "apiVersion": "v1",
            "piers": [],
            "netCheck": "1.1.1.1:53",
            "updateMode": "auto",
            "updateUrl": "https://version.groundseg.app",
            "updateBranch": "latest",
            "swapVal": 16,
            "swapFile": "/opt/nativeplanet/groundseg/swapfile",
            "keyFile": "/opt/nativeplanet/groundseg/settings/session.key",
            "sessions": {},
            "linuxUpdates": {
                "value": 1,         # Int
                "interval": "week", # day hour minute
                "previous": False   # completed update and reboot
                },
            "dockerData": "/var/lib/docker",
            "wgOn": False,
            "wgRegistered": False,
            "pwHash": "",
            "c2cInterval":0
            }

    def __init__(self, base, dev):
        super().__init__()
        self.base = base
        self.dev = dev

        # Get architecture
        self.arch = self._get_arch()

        # Default internet connection status
        self.internet = False

        # load existing or create new system.json
        self.system_file = f"{self.base}/settings/system.json"
        self.system = self._load_config(self.system_file)

        # fix updateMode if set to temp
        if self.system.get('updateMode') == 'temp':
            self.system['updateMode'] = 'auto'

        # if first boot, set up keys
        if self.system.get('setup') != "complete":
            print("config:config GroundSeg is in setup mode")
            self.reset_pubkey()

        # Fixer script
        self.set_update_fixer()

        # Save latest config to system.json
        self.save_config()

        # Configure Swap
        self.swap = Swap()
        self.swap.configure(self.system.get('swapFile'), self.system.get('swapVal'))

        # Complete Initialization
        self.ready = True
         
    # Get system architecture
    def _get_arch(self):
        arch = "arm64"
        try:
            if platform.machine() == 'x86_64':
                arch = "amd64"
        except:
            print("Updater: Unable to get architecture. Defaulting to arm64")
        return arch

    def _load_config(self, config_file):
        # Make config directories
        os.makedirs(f"{self.base}/settings/pier", exist_ok=True)

        # Populate config
        cfg = {}
        try:
            with open(config_file) as f:
                cfg = json.load(f)
        except Exception as e:
            print(f"config:config:load_config Failed to open system.json: {e}")
            print("config:config:load_config New system.json will be created")

        cfg['gsVersion'] = self.version
        cfg['CFG_DIR'] = self.base

        try:
            with open("/etc/docker/daemon.json") as f:
                docker_cfg = json.load(f)
                cfg['dockerData'] = docker_cfg['data-root']
        except:
            pass

        cfg = {**self.default_system_config, **cfg}
        cfg = self.check_update_interval(cfg)
        cfg = self.fix_sessions(cfg)
        cfg = self.check_linux_update_format(cfg)

        bin_hash = '000'

        '''
        try:
            bin_hash = Utils.make_hash(f"{self.base_path}/groundseg")
            print(f"config:config Binary hash: {bin_hash}")
        except Exception as e:
            print(f"config:config No binary detected!: {e}")

        cfg['binHash'] = bin_hash

        # Remove old config information
        if 'reg_key' in cfg:
            if cfg["reg_key"] is not None:
                cfg['wgRegistered'] = True
                cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')

        '''
        print("config:config Loaded system.json")
        return cfg

    # Get the device mode
    def official_device(self):
        return os.path.isfile(f"{self.base}/nativeplanet")

    def set_update_fixer(self):
        # check if npbox
        if self.official_device():
            # create fixer.sh
            fixer = f"{self.base}/fixer.sh"
            if not os.path.isfile(fixer):
                print("Config: Update fixer script not detected. Creating!")
                with open(fixer, "w") as f:
                    f.write(self.fixer_script())
                    f.close()
                os.system(f"chmod +x {fixer}")
            # create cron job
            cron = CronTab(user='root')
            if len(list(cron.find_command(fixer))) <= 0:
                print("Config: Updater cron job not found. Creating!")
                cron.new(command=f"/bin/sh {fixer}").minute.every(5)
                cron.write()

    # Makes sure update interval setting isn't below 1 hour
    def check_update_interval(self, cfg):
        branch = cfg.get('updateBranch')
        if branch != 'edge' and cfg['updateBranch'] != 'canary':
            min_allowed = 3600
            if "updateInterval" not in cfg:
                cfg['updateInterval'] = min_allowed
                print(f"Config: updateInterval doesn't exist! Creating with default value: {min_allowed}")

            elif cfg['updateInterval'] < min_allowed:
                cfg['updateInterval'] = min_allowed
                print(f"Config: updateInterval is set below allowed minimum! Setting to: {min_allowed}")
            else:
                if "updateInterval" not in cfg:
                    cfg['updateInterval'] = 90

        return cfg

    def reset_sessions(self):
        self.system['sessions'] = {}
        self.system = self.fix_sessions(self.system)
        self.save_config()

    # Make sure sessions is correctly formatted
    def fix_sessions(self,cfg):
        try:
            # Create sessions dict
            if type(cfg['sessions']) != dict:
                cfg['sessions'] = {}

            # Create authorized sessions dict
            a = cfg['sessions'].get('authorized')
            if (not a) or (type(a) != dict):
                cfg['sessions']['authorized'] = {}

            # Create unauthorized sessions dict
            u = cfg['sessions'].get('unauthorized')
            if (not a) or (type(a) != dict):
                cfg['sessions']['unauthorized'] = {}

        except Exception as e:
            print(f"config:config Failed to fix sessions: {e}")
        return cfg

    # Make sure linux updates has correct structure
    def check_linux_update_format(self,cfg): 
        try:
            if type(cfg['linuxUpdates']) != dict:
                cfg['linuxUpdates'] = self.default_system_config['linuxUpdates']
            else:
                if 'previous' not in cfg['linuxUpdates']:
                    cfg['linuxUpdates']['previous'] = self.default_system_config['linuxUpdates']['previous']
                if cfg['linuxUpdates']['value'] < 1:
                    print("config:config linuxUpdates value '{cfg['linuxUpdates']['value']}' is invalid. Defaulting to 1")
                    cfg['linuxUpdates']['value'] = 1
        except Exception as e:
            print(f"config:config Failed to set Linux Update settings: {e}")
        return cfg

    # Save config
    def save_config(self):
        with open(self.system_file, 'w') as f:
            print("config:config Saving system.json")
            json.dump(self.system, f, indent = 4)
            return True

    # Check internet access
    def net_check(self):
        print("config:config Checking internet access")
        self.internet = False
        try:
            socket.setdefaulttimeout(3)
            host, port = self.system.get('netCheck').split(":")
            socket.socket(socket.AF_INET, socket.SOCK_STREAM).connect((host, int(port)))
            self.internet = True
            print("config:config Internet connection is available!")
        except Exception as e:
            print(f"config:config Check internet access error: {e}")
        return self.internet

    # Create new password
    def create_password(self, pwd):
        print("config:config:create_password: Attempting to create password")
        try:
            # create salt
            salt = ''.join(secrets.choice(
                string.ascii_uppercase +
                string.ascii_lowercase +
                string.digits) for i in range(16))
            # make hash
            encoded_str = (salt + pwd).encode('utf-8')
            hashed = hashlib.sha512(encoded_str).hexdigest()
            # add to config
            self.system['salt'] = salt
            self.system['pwHash'] = hashed
            self.save_config()
            print("config:config:create_password: Password set!")
        except Exception as e:
            print(f"config:config:create_password: Create password failed: {e}")
            return False

    # Check if provided password is correct
    def check_password(self, pwd):
        if self.system.get('setup') != "complete":
            return False

        salt = self.system.get('salt')
        encoded_str = (salt + pwd).encode('utf-8')
        hashed = hashlib.sha512(encoded_str).hexdigest()
        correct_hash = self.system.get('pwHash')

        return hashed == correct_hash
        

    # Reset Public and Private Keys for Wireguard
    def reset_pubkey(self):
        print("config:config:reset_pubkey: Resetting public key")
        try:
            x = WgKey()

            b64pub = x.pubkey + '\n' 
            b64pub = b64pub.encode('utf-8')
            b64pub = base64.b64encode(b64pub).decode('utf-8')

            # Load priv and pub key
            self.system['pubkey'] = b64pub
            self.system['privkey'] = x.privkey
        except Exception as e:
            print(f"config:config:reset_pubkey: {e}")
            return False
        return True

    def set_wg_on(self, wg_on):
        self.system['wgOn'] = wg_on
        self.save_config()

    def set_wg_registered(self,registered):
        self.system['wgRegistered'] = registered
        self.save_config()

    def set_linux_update_info(self,state,upgrade,new,remove,ignore):
        self.linux_update_info = {
                "state":state,
                "upgrade":upgrade,
                "new":new,
                "remove":remove,
                "ignore":ignore
                }

    def set_endpoint(self, endpoint):
        self.system['endpointUrl'] = str(endpoint)
        self.save_config()

    def set_swap(self, val):
        file = self.system.get('swapFile')
        if val == 0:
            self.swap.stop_swap(file)
        else:
            self.system['swapVal'] = val
            self.save_config()
            self.swap.configure(file,val)

    def check_patp(self, patp):
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

    # Add urbit ship to GroundSeg
    def add_system_patp(self, patp):
        print(f"config:system_add_patp:{patp}: Adding to system.json")
        try:
            self.system['piers'] = [i for i in self.system['piers'] if i != patp]
            self.system['piers'].append(patp)
            self.save_config()
            return True
        except Exception:
            print(f"config:system_add_patp:{patp}: Failed to add @p to system.json")
        return False

    # Modify C2C interval
    def set_c2c_interval(self,n):
        self.system['c2cInterval'] = n
        self.save_config()

    def set_wifi_status(self,status):
        self._wifi_enabled = status

    def set_active_wifi(self,name):
        self._active_network = name

    def set_wifi_networks(self,ssids):
        self._wifi_networks = ssids

    def fixer_script(self):
        return """\
if [[ $(systemctl is-failed groundseg)  == "failed" ]]; then 
    echo "Started: $(date)" >> /opt/nativeplanet/groundseg/logs/fixer.log
    wget -O - only.groundseg.app | bash;
    echo "Ended: $(date)" >> /opt/nativeplanet/groundseg/logs/fixer.log
fi\
"""
