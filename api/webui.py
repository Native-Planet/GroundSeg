#Python
import json

# GroundSeg modules
from log import Log
from webui_docker import WebUIDocker

class WebUI:

    data = {}
    default_config = { 
                      "webui_name": "groundseg-webui",
                      "webui_version": "latest",
                      "repo": "nativeplanet/groundseg-webui",
                      "port": 80,
                      "background": ""
                      }   

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.filename = f"{config.base_path}/settings/webui.json"
        self.webui_docker = WebUIDocker()

        self.load_config()

        # Set Netdata Config
        self.load_config()
        branch = self.config['updateBranch']
        self.data = {**self.default_config, **self.data}

        # Updater WebUI information
        if (self.config_object.update_avail) and (self.config['updateMode'] == 'auto'):
            Log.log("WebUI: Replacing local data with version server data")
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['webui']
            self.data['repo'] = self.updater_info['repo']
            self.data['webui_version'] = self.updater_info['tag']
            self.data['amd64_sha256'] = self.updater_info['amd64_sha256']
            self.data['arm64_sha256'] = self.updater_info['arm64_sha256']

        # image replaced by repo
        if 'image' in self.data:
            self.data.pop('image')

        # tag replaced by wireguard_version
        if 'tag' in self.data:
            self.data.pop('tag')

        self.save_config()

        if self.start():
            Log.log("WebUI: Initialization Completed")

    # Start container
    def start(self):
        return self.webui_docker.start(self.data, self.config_object._arch)

    # Load webui.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.data = json.load(f)
                Log.log("WebUI: Successfully loaded webui.json")

        except Exception as e:
            Log.log(f"WebUI: Failed to open webui.json: {e}")
            Log.log("WebUI: New webui.json will be created")

    # Save webui.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()
