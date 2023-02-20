# Python
import os
import ssl
import json
import base64
from time import sleep

# Modules
from pywgkey.key import WgKey

# GroundSeg Modules
from log import Log
from utils import Utils
from hasher import Hash

class Config:
    # System
    _ram = None
    _cpu = None
    _core_temp = None
    _disk = None

    # GroundSeg
    version = "v1.1.0"
    config_file = None
    update_payload = {}
    device_mode = "standard"
    config = None
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
            "webuiPort": "80", #str
            "updateBranch": "latest",
            "updateUrl": "https://version.groundseg.app",
            "c2cInterval": 0 #int
            }


    def __init__(self, base_path):
        # store config file location
        self.base_path = base_path
        self.config_file = f"{self.base_path}/settings/system.json"

        # load existing or create new system.json
        self.config = self.load_config(self.config_file)
        Log.log("Loaded system JSON")

        # if first boot, set up keys
        if self.config['firstBoot']:
            Log.log("GroundSeg is in setup mode")
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
            Log.log(f"Failed to open system.json: {e}")
            Log.log("New system.json will be created")

        except:
            pass

        cfg['gsVersion'] = self.version
        cfg['CFG_DIR'] = self.base_path
        cfg = {**self.default_system_config, **cfg}
        cfg = self.check_update_interval(cfg)

        bin_hash = '000'

        try:
            bin_hash = Hash.make_hash(f"{self.base_path}/groundseg")
            Log.log(f"Binary hash: {bin_hash}")
        except Exception as e:
            print(e)
            Log.log("No binary detected!")

        cfg['binHash'] = bin_hash

        # Remove old config information
        if 'reg_key' in cfg:
            if cfg['reg_key'] != None:
                cfg['wgRegistered'] = True
            cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')
        
        return cfg


    # Makes sure update interval setting isn't below 1 hour
    def check_update_interval(self, cfg):
        if cfg['updateBranch'] != 'edge':
            min_allowed = 3600
            if not 'updateInterval' in cfg:
                cfg['updateInterval'] = min_allowed
                Log.log(f"updateInterval doesn't exist! Creating with default value: {min_allowed}")
            elif cfg['updateInterval'] < min_allowed:
                cfg['updateInterval'] = min_allowed
                Log.log(f"updateInterval is set below allowed minimum! Setting to: {min_allowed}")
        else:
            cfg = self.check_config_field(cfg, 'updateInterval', 90)

        return cfg

    # Reset Public and Private Keys
    def reset_pubkey(self):
        x = WgKey()

        b64pub = x.pubkey + '\n' 
        b64pub = b64pub.encode('utf-8')
        b64pub = base64.b64encode(b64pub).decode('utf-8')

        # Load priv and pub key
        self.config['pubkey'] = b64pub
        self.config['privkey'] = x.privkey


    def set_device_mode(self):
        self.check_mode_file()
        internet = Utils.check_internet_access()
        if self.device_mode == "npbox":
            if not internet:
                Log.log("No internet access, starting Connect to Connect mode")
                self.device_mode == "c2c"
        else:
            while not internet:
                Log.log("No internet access, checking again in 15 seconds")
                sleep(15)
                internet = Utils.check_internet_access()


    def check_mode_file(self):
        if os.path.isfile(f"{self.base_path}/vm") or 'WSL_DISTRO_NAME' in os.environ:
            Log.log("VM mode detected. Enabling limited features")
            self.device_mode = "vm"

        if os.path.isfile(f"{self.base_path}/nativeplanet"):
            Log.log("NP box detected. Enabling NP box features")
            self.device_mode = "npbox"


    def save_config(self):
        with open(self.config_file, 'w') as f:
            json.dump(self.config, f, indent = 4)
