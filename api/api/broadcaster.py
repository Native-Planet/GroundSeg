class Broadcaster:
    def __init__(self,cfg):
        self.cfg = cfg

    def broadcast(self):
        broadcast = {
                "type": "structure",
                "auth-level": "authorized",
                "system": {
                    "usage": {
                        "ram": self.cfg._ram,
                        "cpu": self.cfg._cpu,
                        "cpu_temp": self.cfg._core_temp,
                        "disk": self.cfg._disk
                        }
                    }
                }
        print(broadcast)

    def setup(self):
        broadcast = {
                "type": "structure",
                "auth_level": "setup",
                "stage": "start",
                "page": 0
               }
        #print(broadcast)
