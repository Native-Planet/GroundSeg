import asyncio
import websockets

class WS:
    def __init__(self, groundseg, host, port, dev):
        super().__init__()
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

    async def run(self):
        server = await websockets.serve(self.handler, self.host, self.port)
        await server.wait_closed()

