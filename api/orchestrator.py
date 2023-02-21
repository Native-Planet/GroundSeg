
# GroundSeg modules
from log import Log
from wireguard import Wireguard

class Orchestrator:

    wireguard = None

    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.wireguard = Wireguard(config)
        #self.minio
        #self.urbit
        #self.webui

        self.config_object.gs_ready = True
        Log.log("Initialization completed")
