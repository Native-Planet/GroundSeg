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
                      "image": "nativeplanet/groundseg-webui",
                      "tag": "latest",
                      "port": 80
                      }   

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.filename = f"{config.base_path}/settings/webui.json"
        self.webui_docker = WebUIDocker()

        self.load_config()

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("WebUI: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater WebUI information
        if self.config_object.update_avail:
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['webui']
            self.data['image'] = self.updater_info['repo']
            self.data['tag'] = self.updater_info['tag']
        self.data = {**self.default_config, **self.data}

        self.save_config()

        if self.start():
            Log.log("WebUI: Initialization Completed")

    # Start container
    def start(self):
        return self.webui_docker.start(self.data,self.updater_info, self.config_object._arch)

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
