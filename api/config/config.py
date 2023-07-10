import os
import json
import socket
import platform

from config.swap import Swap

class Config:
    # GroundSeg version
    version = "v2.0.0"

    # Class is ready
    ready = False

    # default content of system.json
    default_system_config = {
            "setup": True,
            "netCheck": "1.1.1.1:53",
            "updateMode": "auto",
            "swapVal": 16,
            "swapFile": "/opt/nativeplanet/groundseg/swapfile",
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
        if self.system.get('setup'):
            print("config:config GroundSeg is in setup mode")
            #self.reset_pubkey()

        # Save latest config to system.json
        self.save_config()

        # Configure Swap
        Swap().configure(self.system.get('swapFile'), self.system.get('swapVal'))

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
            print(f"config:config Failed to open system.json: {e}")
            print("config:config New system.json will be created")

        cfg['gsVersion'] = self.version
        cfg['CFG_DIR'] = self.base

        try:
            with open("/etc/docker/daemon.json") as f:
            docker_cfg = json.load(f)
            cfg['dockerData'] = docker_cfg['data-root']
        except:
            pass

        cfg = {**self.default_system_config, **cfg}
        #cfg = self.check_update_interval(cfg)

        '''
        try:
            if type(cfg['sessions']) != dict:
            cfg['sessions'] = {}

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

        bin_hash = '000'

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

        print("config:config Loaded system.json")
        '''

        return cfg

    # Save config
    def save_config(self):
        with open(self.system_file, 'w') as f:
            print("config:config Saving system.json")
            json.dump(self.system, f, indent = 4)
            return True

    # Check internet access
    async def net_check(self):
        print("config:config Checking internet access")
        try:
            socket.setdefaulttimeout(3)
            host, port = self.system.get('netCheck').split(":")
            socket.socket(socket.AF_INET, socket.SOCK_STREAM).connect((host, int(port)))
            self.internet = True
            print("config:config Internet connection is available!")
            return
        except Exception as e:
            Log.log(f"config:config Check internet access error: {e}")
        self.internet = False
        return
