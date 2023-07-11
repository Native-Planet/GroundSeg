class Broadcaster:
    def __init__(self,cfg):
        self.cfg = cfg

    def broadcast(self):
        broadcast = {
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

    def unready(self):
        print("NOT_READY")
