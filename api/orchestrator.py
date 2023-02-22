
# GroundSeg modules
from log import Log
from wireguard import Wireguard
from urbit import Urbit
from webui import WebUI

class Orchestrator:

    wireguard = None

    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.wireguard = Wireguard(config)
        self.urbit = Urbit(config)
        #self.minio
        self.webui = WebUI(config)

        self.config_object.gs_ready = True
        Log.log("GroundSeg: Initialization completed")

    # List of Urbit Ships in Home Page
    def get_urbits(self):
        return self.urbit.list_ships()
