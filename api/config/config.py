import os
import json
import socket
import platform

class Config:
    # default content of system.json
    default_system_config = {
        "setup": True,
        "netCheck": "1.1.1.1:53",
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
        self.system = self._load_config(f"{self.base}/settings/system.json")

        '''
        # fix updateMode if set to temp
        if self.config['updateMode'] == 'temp':
            self.config['updateMode'] = 'auto'

        # if first boot, set up keys
        if self.config['firstBoot']:
            print("Config: First Boot detected! GroundSeg is in setup mode")
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
                print(f"Swap: Removing {self.config['swapFile']}")
                os.remove(self.config['swapFile'])

                if Utils.make_swap(self.config['swapFile'], self.config['swapVal']):
                Utils.start_swap(self.config['swapFile'])

                # Set current mode
                self.set_device_mode()
                self.state['ready']['config'] = True
        '''
         
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
            print(f"Config: Failed to open system.json: {e}")
            print("Config: New system.json will be created")

        '''
        cfg['gsVersion'] = self.version
        cfg['CFG_DIR'] = self.base

        try:
            with open("/etc/docker/daemon.json") as f:
            docker_cfg = json.load(f)
            cfg['dockerData'] = docker_cfg['data-root']
        except:
            pass
        '''

        cfg = {**self.default_system_config, **cfg}
        '''
        cfg = self.check_update_interval(cfg)

        try:
            if type(cfg['sessions']) != dict:
            cfg['sessions'] = {}

            if type(cfg['linuxUpdates']) != dict:
                cfg['linuxUpdates'] = self.default_system_config['linuxUpdates']
            else:
                if 'previous' not in cfg['linuxUpdates']:
                    cfg['linuxUpdates']['previous'] = self.default_system_config['linuxUpdates']['previous']
                if cfg['linuxUpdates']['value'] < 1:
                    print("Config: linuxUpdates value '{cfg['linuxUpdates']['value']}' is invalid. Defaulting to 1")
                    cfg['linuxUpdates']['value'] = 1
        except Exception as e:
            print(f"Config: Failed to set Linux Update settings: {e}")

        bin_hash = '000'

        try:
            bin_hash = Utils.make_hash(f"{self.base_path}/groundseg")
            print(f"Config: Binary hash: {bin_hash}")
        except Exception as e:
            print(f"Config: No binary detected!: {e}")

        cfg['binHash'] = bin_hash

        # Remove old config information
        if 'reg_key' in cfg:
            if cfg["reg_key"] is not None:
                cfg['wgRegistered'] = True
                cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')

        print("Config: Loaded system.json")
        '''

        return cfg

    async def net_check(self):
        print("Config: Checking internet access")
        try:
            socket.setdefaulttimeout(3)
            host, port = self.system.get('netCheck').split(":")
            socket.socket(socket.AF_INET, socket.SOCK_STREAM).connect((host, int(port)))
            self.internet = True
            return
        except Exception as e:
            Log.log(f"Config: Check internet access error: {e}")
        self.internet = False
        return
