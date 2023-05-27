import json
import asyncio
from threading import Thread
from websockets.server import serve

from orchestrator import Orchestrator

class GroundSeg:
    def __init__(self):
        self.orchestrator = None
        self.state = {
                "debug":False,
                "ready": False,
                "host": '0.0.0.0',
                "port": '8000',
                "broadcast": {},
                "personal_broadcast": {},
                "tokens": {},
                "dockers": {},
                "configs": {},
                "clients": {
                    "authorized": {},
                    "unauthorized": {}
                    }
                }

    def run(self):
        # start orchestrator in bg
        Thread(target=self.init_orchestrator).start()
        # start websocket
        asyncio.run(self.serve())

    def init_orchestrator(self):
        self.orchestrator = Orchestrator(self.state)

    async def handle(self, websocket):
        print("app:handle Message received")
        try:
            async for message in websocket:
                try:
                    ready = self.orchestrator.state.get('ready')
                except:
                    ready = False
                if ready:
                    action = json.loads(message)
                    activity = self.orchestrator.handle_request(action, websocket)
                    await websocket.send(activity)
                else:
                    # send not ready message
                    await websocket.send("NOT READY")

        except Exception as e:
            print(f"app:handle Error {e}")

    async def serve(self):
        async with serve(self.handle, self.state.get('host'), self.state.get('port')):
            # Broadcast here
            '''
            from broadcast import Broadcast
            b = Broadcast(
                    self.ws_util.authorized_clients,
                    self.ws_util.unauthorized_clients,
                    self.ws_util
                    )
            asyncio.get_event_loop().create_task(b.authorized())
            asyncio.get_event_loop().create_task(b.unauthorized())
            '''
            await asyncio.Future()

groundseg = GroundSeg()
groundseg.run()
