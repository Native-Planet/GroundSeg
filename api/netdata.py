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
                      "netdata_version": "latest",
                      "volume_dir": "/var/lib/docker/volumes",
                      "image": "netdata/netdata",
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

        self.load_config()

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("Netdata: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater Netdata information
        if self.config_object.update_avail:
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['netdata']
            self.data['image'] = self.updater_info['repo']
        self.data = {**self.default_config, **self.data}

        self.save_config()
        if self.start():
            Log.log("Netdata: Initialization Completed")

    def start(self):
        return self.nd_docker.start(self.data, self.updater_info, self.config_object._arch)

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
