import asyncio

class Lick:
    def __init__(self, groundseg, dev):
        super().__init__()
        self.app = groundseg
        self.dev = dev

    async def run(self):
        while True:
            try:
                if self.app.ready:
                    #print("updating ships via lick")
                    pass
                else:
                    print("gs not ready")
            except Exception as e:
                print(f"lick ipc error {e}")
            await asyncio.sleep(1)
