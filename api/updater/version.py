import asyncio
import requests

class VersionServer:
    def __init__(self,cfg,dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    async def check(self):
        while True:
            try:
                print("updater:version:check Fetching updated information from version server")
                url = self.cfg.system.get('updateUrl')
                r = requests.get(url)
                if r.status_code == 200:
                    self.cfg.version_server_ready = True
                    self.cfg.version_info = r.json()
                else:
                    raise ValueError(f"Status code {r.status_code}")

            except Exception as e:
                self.cfg.version_server_ready = False
                print(f"updater:version:check Unable to retrieve update information: {e}")
            await asyncio.sleep(60)
