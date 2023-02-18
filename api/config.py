import os
import json

from log import Log
from hasher import Hash

class Config:
    version = "v1.1.0"
    config_file = None
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
        self.save_config()


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

    def save_config(self):
        with open(self.config_file, 'w') as f:
            json.dump(self.config, f, indent = 4)
