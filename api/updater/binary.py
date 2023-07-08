import asyncio

class BinUpdate:
    def __init__(self, cfg, dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    async def run(self):
        while True:
            try:
                print("temp: checking for binary updates")
            except Exception as e:
                print("BinUpdate error {e}")
            await asyncio.sleep(10)
