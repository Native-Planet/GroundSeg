# Python
import os
import ssl
import json
import base64
import string
import secrets
import hashlib
import platform

from time import sleep

# Modules
from pywgkey.key import WgKey

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
    version = "v1.1.1"

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
            "c2cInterval": 0
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

        # if first boot, set up keys
        if self.config['firstBoot']:
            Log.log("Config: First Boot detected! GroundSeg is in setup mode")
            self.reset_pubkey()

        # Save latest config to system.json
        self.save_config()

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
        cfg = {**self.default_system_config, **cfg}
        cfg = self.check_update_interval(cfg)

        bin_hash = '000'

        try:
            bin_hash = Utils.make_hash(f"{self.base_path}/groundseg")
            Log.log(f"Config: Binary hash: {bin_hash}")
        except Exception as e:
            print(e)
            Log.log("Config: No binary detected!")

        cfg['binHash'] = bin_hash

        # Remove old config information
        if 'reg_key' in cfg:
            if cfg['reg_key'] != None:
                cfg['wgRegistered'] = True
            cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')
        
        Log.log("Config: Loaded system.json")

        return cfg


    # Makes sure update interval setting isn't below 1 hour
    def check_update_interval(self, cfg):
        if cfg['updateBranch'] != 'edge':
            min_allowed = 3600
            if not 'updateInterval' in cfg:
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

    
    def change_password(self, data):
        encoded_str = (self.config['salt'] + data['old-pass']).encode('utf-8')
        this_hash = hashlib.sha512(encoded_str).hexdigest()

        Log.log("Config: Attempting to change password")

        if this_hash == self.config['pwHash']:
            if self.create_password(data['new-pass']):
                return True

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
        except Exception as e:
            Log.log("Config: Create password failed: {e}")
            return False

        return True


    def set_device_mode(self):
        Log.log("Config: Setting device mode")
        self.check_mode_file()
        internet = Utils.check_internet_access()
        if self.device_mode == "npbox":
            if not internet:
                Log.log("Config: No internet access, starting Connect to Connect mode")
                self.device_mode == "c2c"
        else:
            while not internet:
                Log.log("Config: No internet access, checking again in 15 seconds")
                sleep(15)
                internet = Utils.check_internet_access()


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
