import asyncio
import websockets

from api.broadcaster import Broadcaster

class WS:
    def __init__(self, cfg, groundseg, host, port, dev):
        super().__init__()
        self.cfg = cfg
        self.app = groundseg
        self.dev = dev
        self.host = host
        self.port = port

    async def handler(self, websocket, path):
        while True:
            message = await websocket.recv()
            if self.app.ready:
                print(f"< {message}")
            else:
                print("< gs not ready")

    async def broadcast(self):
        b = Broadcaster(self.cfg)
        while True:
            if self.app.ready:
                b.broadcast()
            else:
                b.unready()
            await asyncio.sleep(10)


    async def run(self):
        server = await websockets.serve(self.handler, self.host, self.port)
        await server.wait_closed()

