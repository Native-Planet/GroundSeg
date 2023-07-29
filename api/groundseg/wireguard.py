import json
import base64
import requests
from time import sleep

# GroundSeg modules
#from log import Log
from groundseg.docker.wireguard import WireguardDocker

class Wireguard:

    # Logs are allowed to be streamed?
    allow_logs_stream = False
    #
    # StarTram API headers, to be moved to its own class
    _headers = {"Content-Type": "application/json"}
    # The data in wireguard.json
    data = {}
    # Information for Wireguard from the version server
    version_info = {}
    # 
    register_broadcast_status = None
    anchor_services = {}
    anchor_data = {}
    region_data = {}
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

    def __init__(self, cfg):
        self.cfg = cfg
        # This is the wireguard config file
        self.filename = f"{self.cfg.base}/settings/wireguard.json"
        # This is the volume directory. In the future, we will extend it
        # to accept any location set by the user
        self._volume_directory = f"{self.cfg.system.get('dockerData')}/volumes"
        # The docker API wrapper
        self.wg_docker = WireguardDocker() # Docker incomplete for now

        # Initialized the Wireguard Config
        self.load_config()
        self.data = {**self.default_config, **self.data}

        # Get the latest information from the version server class
        branch = self.cfg.system.get('updateBranch')
        if self.cfg.version_server_ready and self.cfg.system.get('updateMode') == 'auto':
            print("groundseg:wireguard:init: Replacing local data with version server data")
            self.version_info = self.cfg.version_info['groundseg'][branch]['wireguard']
            self.data['repo'] = self.version_info['repo']
            self.data['wireguard_version'] = self.version_info['tag']
            self.data['amd64_sha256'] = self.version_info['amd64_sha256']
            self.data['arm64_sha256'] = self.version_info['arm64_sha256']

        # Legacy keys in the config that is no longer used
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

        # Save the updated config
        self.save_config()
        # Start the container if startram is registered and set to on
        if self.cfg.system.get('wgOn') and self.cfg.system.get('wgRegistered'):
            self.start()

        print("groundseg:wireguard:init: Initialization Completed")

    # Start container
    def start(self):
        if self.wg_docker.start(self.data, self.cfg.arch):
            self.cfg.set_wg_on(True)
            return True
        return False

    # Stop container
    def stop(self):
        self.cfg.set_wg_on(False)
        return self.wg_docker.stop(self.data)

    # Remove container and volume
    def remove(self):
        return self.wg_docker.remove_wireguard(self.data['wireguard_name'])

    # Is container running
    def is_running(self):
        if self.cfg.system.get('wgRegistered'):
            return self.wg_docker.is_running(self.data['wireguard_name'])
        return False

    # Update wg0.confg
    def write_wg_conf(self):
        try:
            conf = self.anchor_data.get('conf')
            conf = base64.b64decode(conf).decode('utf-8')
            conf = conf.replace('privkey', self.cfg.system.get('privkey'))
            return self.wg_docker.add_config(self._volume_directory, self.data, conf)
        except Exception as e:
            print(f"groundseg:wireguard:write_wg_conf Failed to write wg0.confg: {e}")


    # wgOn False
    def off(self, urb, minio):
        for p in urb._urbits:
            if urb._urbits[p]['network'] == 'wireguard':
                urb.toggle_network(p)
                minio.stop_all()
                minio.stop_mc()
                self.stop()
                self.config['wgOn'] = False
                self.config_object.save_config()
        return 200

    # Takes list of subdomains from startram and returns a dict
    def get_subdomains(self):
        res = {}
        patp = ''
        subs = self.anchor_data.get('subdomains')
        for sub in subs:
            fragments = sub.get('url').split('.')
            for f in fragments:
                if self.cfg.check_patp(f):
                    if f not in res:
                        res[f] = {}
                    patp = f
                    break
            svc = sub.get('svc_type')
            res[patp][svc] = {
                    "status":sub.get('status'),
                    "url":sub.get('url'),
                    "port":sub.get('port'),
                    "alias":sub.get('alias'),
                    }
            self.anchor_services = res
        return True

    '''
def restart(self, urb, minio):
try:
print("Wireguard: Attempting to restart wireguard")
self.config_object.anchor_ready = False
print("Anchor: Refresh loop is unready")
remote = []
for patp in urb._urbits:
if urb._urbits[patp]['network'] != 'none':
remote.append(patp)

if self.off(urb, minio) == 200:
if self.on(minio) == 200:
if len(remote) <= 0:
return 200
for patp in remote:
urb.toggle_network(patp)
print("Anchor: Refresh loop is ready")
self.config_object.anchor_ready = True
return 200
except Exception as e:
print(f"Wireguard: Failed to restart wireguard: {e}")

return 400

# Container logs
def logs(self):
return self.wg_docker.full_logs(self.data['wireguard_name'])

# New anchor registration
def build_anchor(self, url, reg_key, region):
print("Wireguard: Attempting to build anchor")
try:
if self.register_device(url, reg_key, region):
if self.get_status(url):
if self.start():
if self.update_wg_config(self.anchor_data['conf']):
print("Anchor: Registered with anchor server")
return True

except Exception as e:
print(f"Wireguard: Failed to build anchor: {e}")

return False

# Change Anchor endpoint URL
def change_url(self, url, urb, minio):
print(f"Wireguard: Attempting to change endopint url to {url}")
endpoint = self.config['endpointUrl']
api_version = self.config['apiVersion']
old_url = f'https://{endpoint}/{api_version}'
self.config['endpointUrl'] = url
self.config['wgRegistered'] = False
self.config['wgOn'] = False

for patp in self.config['piers']:
self.delete_service(f'{patp}','urbit',old_url)
self.delete_service(f's3.{patp}','minio',old_url)

self.off(urb, minio)
self.config_object.reset_pubkey()
print("Wireguard: Changed url")
self.config_object.save_config()
if self.config['endpointUrl'] == url:
self.region_data = {}
self.anchor_data = {}
self.get_regions(f"https://{url}/{api_version}")
return 200
return 400
    '''

    # Get log stream status
    def is_stream_allowed(self):
        if self.allow_logs_stream:
            return "open"
        else:
            return "closed"

    def toggle_log_stream(self):
        self.allow_logs_stream = not self.allow_logs_stream

    # Container logs
    def logs(self):
        logs = []
        if self.allow_logs_stream:
            name = self.data.get('wireguard_name')
            try:
                logs = self.wg_docker.wg_show(name).output.decode("utf-8").split("\n")
            except Exception as e:
                print(e)
        return logs

    # Load wireguard.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.data = json.load(f)
            print("Wireguard: Successfully loaded wireguard.json")
        except Exception as e:
            print(f"groundseg:wireguard:load_config: Failed to open wireguard.json: {e}")
            print("groundseg:wireguard:load_config: New wireguard.json will be created")

    # Save wireguard.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()
