# Python
import json

# GroundSeg modules
from log import Log
from netdata_docker import NetdataDocker

class Netdata:
    data = {}
    updater_info = {}
    default_config = {
            "netdata_name": "netdata",
            "repo": "registry.hub.docker.com/netdata/netdata",
            "netdata_version": "latest",
            "amd64_sha256": "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
            "arm64_sha256": "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
            "volume_dir": "/var/lib/docker/volumes",
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

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.nd_docker = NetdataDocker()
        self.filename = f"{self.config_object.base_path}/settings/netdata.json"

        # Set Netdata Config
        self.load_config()
        branch = self.config['updateBranch']
        self.data = {**self.default_config, **self.data}

        # Updater Netdata information
        if (self.config_object.update_avail) and (self.config['updateMode'] == 'auto'):
            Log.log("Netdata: Replacing local data with version server data")
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['netdata']
            self.data['repo'] = self.updater_info['repo']
            self.data['netdata_version'] = self.updater_info['tag']
            self.data['amd64_sha256'] = self.updater_info['amd64_sha256']
            self.data['arm64_sha256'] = self.updater_info['arm64_sha256']

        # image replaced by repo
        if 'image' in self.data:
            self.data.pop('image')

        self.save_config()

        if self.start():
            Log.log("Netdata: Initialization Completed")

    def start(self):
        return self.nd_docker.start(self.data, self.config_object._arch)

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
                Log.log("Netdata: Successfully loaded netdata.json")

        except Exception as e:
            Log.log(f"Netdata: Failed to open netdata.json: {e}")
            Log.log("Netdata: New netdata.json will be created")

    # Save netdata.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()
