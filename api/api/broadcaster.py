import json

class Broadcaster:
    def __init__(self,cfg,groundseg):
        self.cfg = cfg
        self.app = groundseg

    async def broadcast(self):
        a_broadcast = {
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
        u_broadcast = {
                "type": "structure",
                "auth-level": "unauthorized",
                "login": {
                    "info": "temp"
                    }
                }
        await self.authed(a_broadcast)
        await self.unauth(u_broadcast)

    async def setup(self):
        broadcast = {
                "type": "structure",
                "auth_level": "setup",
                "stage": self.app.setup.stage,
                "page": self.app.setup.page
               }
        await self.authed(broadcast)
        await self.unauth(broadcast)

    async def authed(self, broadcast):
        sesh = self.app.active_sessions
        a = sesh.get('authorized').copy().keys()
        for s in a:
            try:
                await s.send(json.dumps(broadcast))
            except:
                print(f"removing {s}")
                self.app.active_sessions['authorized'].pop(s)

    async def unauth(self, broadcast):
        sesh = self.app.active_sessions
        u = sesh.get('unauthorized').copy().keys()
        for s in u:
            try:
                await s.send(json.dumps(broadcast))
            except:
                print(f"removing {s}")
                self.app.active_sessions['unauthorized'].pop(s)
