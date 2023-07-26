# Python
import json

# GroundSeg modules
#from log import Log
from groundseg.docker.netdata import NetdataDocker

class Netdata:
    data = {}
    updater_info = {}
    default_config = {
        "netdata_name": "netdata",
        "repo": "registry.hub.docker.com/netdata/netdata",
        "netdata_version": "latest",
        "amd64_sha256": "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
        "arm64_sha256": "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
        "cap_add": ["SYS_PTRACE"],
        "port": 19999,
        "restart": "unless-stopped",
        "security_opt": "apparmor=unconfined",
        "volumes": [
            "netdataconfig:/etc/netdata",
            "netdatalib:/var/lib/netdata",
            "netdatacache:/var/cache/netdata",
            "/etc/passwd:/host/etc/passwd:ro",
            "/etc/group:/host/etc/group:ro",
            "/proc:/host/proc:ro",
            "/sys:/host/sys:ro",
            "/etc/os-release:/host/etc/os-release:ro"
            ]
        }
        #"volume_dir": "/var/lib/docker/volumes",

    def __init__(self, cfg):
        self.cfg = cfg
        self.nd_docker = NetdataDocker()
        self.filename = f"{self.cfg.base}/settings/netdata.json"

        # Set Netdata Config
        self.load_config()
        self.data = {**self.default_config, **self.data}

        # Get the latest information from the version server class
        branch = self.cfg.system.get('updateBranch')
        if self.cfg.version_server_ready and self.cfg.system.get('updateMode') == 'auto':
            print("groundseg:netdata:init: Replacing local data with version server data")
            self.version_info = self.cfg.version_info['groundseg'][branch]['netdata']
            self.data['repo'] = self.version_info['repo']
            self.data['netdata_version'] = self.version_info['tag']
            self.data['amd64_sha256'] = self.version_info['amd64_sha256']
            self.data['arm64_sha256'] = self.version_info['arm64_sha256']

        # Legacy keys in the config that is no longer used
        # image replaced by repo
        if 'image' in self.data:
            self.data.pop('image')
        # remove volume directory path
        if 'volume_dir' in self.data:
            self.data.pop('volume_dir')

        # Save the updated config
        self.save_config()
        # Start the container if startram is registered and set to on
        if self.start():
            print("groundseg:netdata:init: Initialization Completed")

    def start(self):
        return self.nd_docker.start(self.data, self.cfg.arch)

    # Container logs
    def logs(self):
        return self.nd_docker.full_logs(self.data['netdata_name'])

    #def stop(self):
    #    return self.nd_docker.stop()

    # Load netdata.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.data = json.load(f)
                print("groundseg:netdata:load_config: Successfully loaded netdata.json")
        except Exception as e:
            print(f"groundseg:netdata:load_config: Failed to open netdata.json: {e}")
            print("groundseg:netdata:load_config: New netdata.json will be created")

    # Save netdata.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()
