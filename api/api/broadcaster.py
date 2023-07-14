import json

class Broadcaster:
    def __init__(self,cfg,groundseg):
        self.cfg = cfg
        self.app = groundseg

    async def broadcast(self):
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

    async def setup(self):
        sesh = self.app.active_sessions
        a = sesh.get('authorized').copy()
        u = sesh.get('unauthorized').copy()
        broadcast = {
                "type": "structure",
                "auth_level": "setup",
                "stage": self.app.setup.stage,
                "page": self.app.setup.page
               }
        for s in a:
            try:
                await s.send(json.dumps(broadcast))
            except:
                self.app.active_sessions['authorized'].remove(s)
        for s in u:
            try:
                await s.send(json.dumps(broadcast))
            except:
                self.app.active_sessions['unauthorized'].remove(s)
