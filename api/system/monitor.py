import asyncio

class SysMonitor:
    def __init__(self, cfg, dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    async def ram(self):
        while True:
            try:
                pass
            except Exception as e:
                print("ram error {e}")
            await asyncio.sleep(1)

    async def cpu(self):
        while True:
            try:
                pass
            except Exception as e:
                print("cpu error {e}")
            await asyncio.sleep(1)

    async def temp(self):
        while True:
            try:
                pass
            except Exception as e:
                print("temperature error {e}")
            await asyncio.sleep(1)

    async def disk(self):
        while True:
            try:
                pass
            except Exception as e:
                print("disk error {e}")
            await asyncio.sleep(1)
