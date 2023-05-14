import json
import base64
import requests
from time import sleep

# GroundSeg modules
from log import Log
from wireguard_docker import WireguardDocker

class Wireguard:

    _headers = {"Content-Type": "application/json"}
    data = {}
    updater_info = {}
    default_config = {
            "wireguard_name": "wireguard",
            "wireguard_version": "latest",
            "repo": "registry.hub.docker.com/linuxserver/wireguard",
            "amd64_sha256": "ae6f8e8cc1303bc9c0b5fa1b1ef4176c25a2c082e29bf8b554ce1196731e7db2",
            "arm64_sha256": "403d741b1b5bcf5df1e48eab0af8038355fae3e29419ad5980428f9aebd1576c",
            "cap_add": ["NET_ADMIN","SYS_MODULE"],
            "volumes": ["/lib/modules:/lib/modules"],
            "sysctls": { "net.ipv4.conf.all.src_valid_mark": 1 }
            }

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.filename = f"{self.config_object.base_path}/settings/wireguard.json"
        self.anchor_data = {}
        self.region_data = {}
        self._volume_directory = f"{self.config['dockerData']}/volumes"
        self.wg_docker = WireguardDocker()

        # Set Wireguard Config
        self.load_config()
        branch = self.config['updateBranch']
        self.data = {**self.default_config, **self.data}

        # Updater Wireguard information
        if (self.config_object.update_avail) and (self.config['updateMode'] == 'auto'):
            Log.log("Wireguard: Replacing local data with version server data")
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['wireguard']
            self.data['repo'] = self.updater_info['repo']
            self.data['wireguard_version'] = self.updater_info['tag']
            self.data['amd64_sha256'] = self.updater_info['amd64_sha256']
            self.data['arm64_sha256'] = self.updater_info['arm64_sha256']

        # image replaced by repo
        if 'image' in self.data:
            self.data.pop('image')

        # tag replaced by wireguard_version
        if 'tag' in self.data:
            self.data.pop('tag')

        # remove patp from wireguard config
        if 'patp' in self.data:
            self.data.pop('patp')

        # remove volume directory path
        if 'volume_dir' in self.data:
            self.data.pop('volume_dir')

        self.save_config()

        # TODO: temporary
        name = self.data['wireguard_name']
        tag = self.data['wireguard_version']
        sha = f"{self.config_object._arch}_sha256"
        image = f"{self.data['repo']}:{tag}"
        if self.wg_docker.create_container(name,image,self.data):
            if self.config['wgOn'] and self.config['wgRegistered']:
                self.start()

        Log.log("Wireguard: Initialization Completed")

    # Start container
    def start(self):
        return self.wg_docker.start(self.data, self.config_object._arch)

    # Stop container
    def stop(self):
        return self.wg_docker.stop(self.data)

    # Remove container and volume
    def remove(self):
        return self.wg_docker.remove_wireguard(self.data['wireguard_name'])

    # Is container running
    # TODO: Remove when no longer needed
    def is_running(self):
        if self.config['wgRegistered']:
            return self.wg_docker.is_running(self.data['wireguard_name'])
        return False

    # Container logs
    def logs(self):
        return self.wg_docker.full_logs(self.data['wireguard_name'])

    # Update wg0.confg
    def update_wg_config(self, conf):
        try:
            conf = base64.b64decode(conf).decode('utf-8')
            conf = conf.replace('privkey', self.config['privkey'])
            return self.wg_docker.add_config(self._volume_directory, self.data, conf)

        except Exception as e:
            Log.log(f"Wireguard: Failed to update wg0.confg: {e}")

    # Container logs
    def logs(self, name):
        return self.wg_docker.full_logs(name)

    # Load wireguard.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.data = json.load(f)
                Log.log("Wireguard: Successfully loaded wireguard.json")

        except Exception as e:
            Log.log(f"Wireguard: Failed to open wireguard.json: {e}")
            Log.log("Wireguard: New wireguard.json will be created")

    # Save wireguard.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()

#
#   StarTram API
#

    # /v1/create/alias
    def handle_alias(self, patp, alias, req_type):
        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f"https://{endpoint}/{api_version}"

        headers = {"Content-Type": "application/json"}

        blob = {
            "subdomain": patp,
            "alias": alias,
            "pubkey": self.config['pubkey']
        }
        if req_type == 'post':
            try:
                response = requests.post(f'{url}/create/alias',json=blob,headers=headers).json()
                Log.log(f"Anchor: Sent alias {alias} creation request for {patp}")
                Log.log(f"Anchor: {response}")
                if response['error'] == 0:
                    return True
            except Exception as e:
                Log.log(f"Anchor: Failed to register alias {alias} for {patp}: {e}")

        elif req_type == 'delete':
            try:
                response = requests.delete(f'{url}/create/alias',json=blob,headers=headers).json()
                Log.log(f"Anchor: Sent alias {alias} deletion request for {patp}")
                Log.log(f"Anchor: {response}")
                if response['error'] == 0:
                    return True
            except Exception as e:
                Log.log(f"Anchor: Failed to delete alias {alias} for {patp}: {e}")

        return False
