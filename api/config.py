# Python
import os
import json
import base64
import string
import secrets
import hashlib
import platform

from time import sleep
from datetime import datetime

# Modules
from pywgkey.key import WgKey
from crontab import CronTab

# GroundSeg Modules
from log import Log
from utils import Utils

class Config:

    # Default Values #

    # System
    _ram = None
    _cpu = None
    _core_temp = None
    _disk = None
    _arch = ""

    # Current version
    version = "v1.3.2"

    # Debug mode
    debug_mode = False

    # Base path
    base_path = ""

    # payload received from version server
    update_payload = {}

    # if updater is working properly
    update_avail = False

    # GroundSeg has completed initialization
    gs_ready = False

    # Anchor ready to check again
    anchor_ready = True

    # which mode is GroundSeg running
    device_mode = "standard"

    # system.json location
    config_file = None

    # system.json contents
    config = {}

    # Login Key Pairs
    login_keys = {"old":{"pub":"","priv":""},"cur":{"pub":"","priv":""}}

    # Login Status
    login_status = {"locked": False, "end": datetime(1,1,1,0,0), "attempts": 0}

    # Upload status
    upload_status = {}

    # default content of system.json
    default_system_config = {
            "firstBoot": True,
            "piers": [],
            "endpointUrl": "api.startram.io",
            "apiVersion": "v1",
            "wgRegistered": False,
            "wgOn": False,
            "updateMode": "auto",
            "sessions": [],
            "pwHash": "",
            "updateBranch": "latest",
            "updateUrl": "https://version.groundseg.app",
            "c2cInterval": 0,
            "netCheck": "1.1.1.1:53",
            "dockerData": "/var/lib/docker",
            "swapFile": "/opt/nativeplanet/groundseg/swapfile",
            "swapVal": 16,
            "linuxUpdates": {
                "value": 1,         # Int
                "interval": "week", # day hour minute
                "previous": False   # completed update and reboot
                }
            }

    def __init__(self, base_path, debug_mode=False):
        self.debug_mode = debug_mode
        # Announce
        if self.debug_mode:
            Log.log("---------- Starting GroundSeg in debug mode ----------")
        else:
            Log.log("----------------- Starting GroundSeg -----------------")
            Log.log("------------------ Urbit is love <3 ------------------")

        # Get architecture
        self._arch = self.get_arch()

        # store config file location
        self.base_path = base_path
        self.config_file = f"{self.base_path}/settings/system.json"

        # load existing or create new system.json
        self.config = self.load_config(self.config_file)

        # fix updateMode if set to temp
        if self.config['updateMode'] == 'temp':
            self.config['updateMode'] = 'auto'

        # if first boot, set up keys
        if self.config['firstBoot']:
            Log.log("Config: First Boot detected! GroundSeg is in setup mode")
            self.reset_pubkey()

        # Save latest config to system.json
        self.save_config()

        # Set swap
        if self.config['swapVal'] > 0:
            if not os.path.isfile(self.config['swapFile']):
                Utils.make_swap(self.config['swapFile'], self.config['swapVal'])

            Utils.start_swap(self.config['swapFile'])
            swap = Utils.active_swap(self.config['swapFile'])

            if swap != self.config['swapVal']:
                if Utils.stop_swap(self.config['swapFile']):
                    Log.log(f"Swap: Removing {self.config['swapFile']}")
                    os.remove(self.config['swapFile'])

                if Utils.make_swap(self.config['swapFile'], self.config['swapVal']):
                    Utils.start_swap(self.config['swapFile'])

        # Set current mode
        self.set_device_mode()


    # Checks if system.json and all its fields exists, adds field if incomplete
    def load_config(self, config_file):
        # Make config directories
        os.makedirs(f"{self.base_path}/settings/pier", exist_ok=True)

        # Populate config
        cfg = {}
        try:
            with open(config_file) as f:
                cfg = json.load(f)
        except Exception as e:
            Log.log(f"Config: Failed to open system.json: {e}")
            Log.log("Config: New system.json will be created")

        cfg['gsVersion'] = self.version
        cfg['CFG_DIR'] = self.base_path

        try:
            with open("/etc/docker/daemon.json") as f:
                docker_cfg = json.load(f)
                cfg['dockerData'] = docker_cfg['data-root']
        except:
            pass

        cfg = {**self.default_system_config, **cfg}
        cfg = self.check_update_interval(cfg)

        try:
            if type(cfg['linuxUpdates']) != dict:
                cfg['linuxUpdates'] = self.default_system_config['linuxUpdates']
            else:
                if 'previous' not in cfg['linuxUpdates']:
                    cfg['linuxUpdates']['previous'] = self.default_system_config['linuxUpdates']['previous']
            if cfg['linuxUpdates']['value'] < 1:
                Log.log("Config: linuxUpdates value '{cfg['linuxUpdates']['value']}' is invalid. Defaulting to 1")
                cfg['linuxUpdates']['value'] = 1
        except Exception as e:
            Log.log(f"Config: Failed to set Linux Update settings: {e}")

        bin_hash = '000'

        try:
            bin_hash = Utils.make_hash(f"{self.base_path}/groundseg")
            Log.log(f"Config: Binary hash: {bin_hash}")
        except Exception as e:
            Log.log(f"Config: No binary detected!: {e}")

        cfg['binHash'] = bin_hash

        # Remove old config information
        if 'reg_key' in cfg:
            if cfg["reg_key"] is not None:
                cfg['wgRegistered'] = True
            cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')
        
        Log.log("Config: Loaded system.json")

        return cfg


    # Makes sure update interval setting isn't below 1 hour
    def check_update_interval(self, cfg):
        if cfg['updateBranch'] != 'edge' or cfg['updateBranch'] != 'canary':
            min_allowed = 3600
            if "updateInterval" not in cfg:
                cfg['updateInterval'] = min_allowed
                Log.log(f"Config: updateInterval doesn't exist! Creating with default value: {min_allowed}")

            elif cfg['updateInterval'] < min_allowed:
                cfg['updateInterval'] = min_allowed
                Log.log(f"Config: updateInterval is set below allowed minimum! Setting to: {min_allowed}")
        else:
            cfg['updateInterval'] = 90

        return cfg

    # Reset Public and Private Keys
    def reset_pubkey(self):
        Log.log("Config: Resetting public key")
        try:
            x = WgKey()

            b64pub = x.pubkey + '\n' 
            b64pub = b64pub.encode('utf-8')
            b64pub = base64.b64encode(b64pub).decode('utf-8')

            # Load priv and pub key
            self.config['pubkey'] = b64pub
            self.config['privkey'] = x.privkey
        except Exception as e:
            Log.log(f"Config: {e}")

    # Modify current password
    def change_password(self, data):
        used = "new"
        Log.log("Config: Attempting to change password with current key")
        decrypted = Utils.decrypt_password(self.login_keys['cur']['priv'], data['old-pass'])
        if Utils.compare_password(self.config['salt'], decrypted, self.config['pwHash']):
            Log.log("Config: Supplied password is correct")
        else:
            Log.log("Config: Attempting to change password with current previous key")
            decrypted = Utils.decrypt_password(self.login_keys['old']['priv'], data['old-pass'])
            if Utils.compare_password(self.config['salt'], decrypted, self.config['pwHash']):
                Log.log("Config: Supplied password is correct")
                used = "old"
            else:
                Log.log("Config: Supplied password is incorrect")
                return False

        if used == "old":
            decrypted = Utils.decrypt_password(self.login_keys['old']['priv'], data['new-pass'])
        else:
            decrypted = Utils.decrypt_password(self.login_keys['cur']['priv'], data['new-pass'])

        if self.create_password(decrypted):
            return True

        Log.log("Config: Failed to change password")
        return False

    # Create new password
    def create_password(self, pwd):
        Log.log("Config: Attempting to create password")
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
            self.config['salt'] = salt
            self.config['pwHash'] = hashed
            self.save_config()
            Log.log("Config: Password set!")
        except Exception:
            Log.log("Config: Create password failed: {e}")
            return False

        return True


    def set_device_mode(self):
        Log.log("Config: Setting device mode")
        self.check_mode_file()
        self.set_update_fixer()
        self.device_mode_internet()


    def set_update_fixer(self):
        # check if npbox
        if self.device_mode == "npbox":
            # create fixer.sh
            fixer = f"{self.base_path}/fixer.sh"
            if not os.path.isfile(fixer):
                Log.log("Config: Update fixer script not detected. Creating!")
                with open(fixer, "w") as f:
                    f.write(self.fixer_script())
                    f.close()
                os.system(f"chmod +x {fixer}")
            # create cron job
            cron = CronTab(user='root')
            if len(list(cron.find_command(fixer))) <= 0:
                Log.log("Config: Updater cron job not found. Creating!")
                cron.new(command=f"/bin/sh {fixer}").minute.every(5)
                cron.write()

    def device_mode_internet(self):
        internet = Utils.check_internet_access(self.config['netCheck'])
        if self.device_mode == "npbox":
            if not internet:
                Log.log("Config: No internet access, starting Connect to Connect mode")
                self.device_mode = "c2c"
        else:
            while not internet:
                Log.log("Config: No internet access, checking again in 15 seconds")
                sleep(15)
                internet = Utils.check_internet_access(self.config['netCheck'])


    def check_mode_file(self):
        if os.path.isfile(f"{self.base_path}/vm") or 'WSL_DISTRO_NAME' in os.environ:
            Log.log("Config: VM mode detected. Enabling limited features")
            self.device_mode = "vm"

        if os.path.isfile(f"{self.base_path}/nativeplanet"):
            Log.log("Config: NP box detected. Enabling NP box features")
            self.device_mode = "npbox"


    # Get system architecture
    def get_arch(self):
        arch = "arm64"
        try:
            if platform.machine() == 'x86_64':
                arch = "amd64"
        except:
            Log.log("Updater: Unable to get architecture. Defaulting to arm64")

        return arch


    # Save config
    def save_config(self):
        with open(self.config_file, 'w') as f:
            Log.log("Config: Saving system.json")
            json.dump(self.config, f, indent = 4)
            return True


    def fixer_script(self):
        return """\
if [[ $(systemctl is-failed groundseg)  == "failed" ]]; then 
    echo "Started: $(date)" >> /opt/nativeplanet/groundseg/logs/fixer.log
    wget -O - only.groundseg.app | bash;
    echo "Ended: $(date)" >> /opt/nativeplanet/groundseg/logs/fixer.log
fi\
"""
