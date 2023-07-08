import asyncio

class LinUpdate:
    def __init__(self, cfg, dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    async def run(self):
        while True:
            try:
                print("temp: checking for linux updates")
            except Exception as e:
                print("LinUpdate error {e}")
            await asyncio.sleep(10)
