class LogsBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.cfg = self.app.cfg

    def display(self):
        return {
                "containers": {
                    "wireguard":{
                        "logs": self.app.wireguard.logs()
                        }
                },
                "system": {
                    "stream":False,
                    "logs": []
                    }
                }
