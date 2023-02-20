
# GroundSeg modules
from log import Log

class Orchestrator:

    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.wireguard()
        self.minio()
        self.urbit()
        self.webui()

        Log.log("Initialization completed")

    def wireguard(self):
        # if legit
        # check if a container called wireguard exists
        print("wireguard start logic")

    def minio(self):
        print("minio start logic")

    def urbit(self):
        print("urbit start logic")

    def webui(self):
        print("start webui logic")
