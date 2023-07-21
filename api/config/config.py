import os
import json
import socket
import string
import secrets
import hashlib
import platform

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

    # Http Upload open
    http_open = False

    # Uploader secret
    upload_secret = ''

    # default content of system.json
    default_system_config = {
            # The setup stages are
            # start -> profile -> startram -> complete
            "setup": "start",
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
            "wgRegisterd": False
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
            print(f"config:config Check internet access error: {e}")
        self.internet = False
        return

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
        except Exception:
            print("config:config:create_password: Create password failed: {e}")
            return False
        return True
