import asyncio
import websockets
import json
from threading import Thread

from log import Log
from websocket_util import WSUtil
#from response_builder import ResponseBuilder

class GSWebSocket(Thread):
    def __init__(self, config, orchestrator, host='0.0.0.0', port=8000):
        super().__init__()
        self.config_class = config
        self.config = config.config
        self.orchestrator = orchestrator
        self.host = host
        self.port = port

    async def handle(self, websocket, path):
        try:
            async for message in websocket:
                data = json.loads(message)
                valid = True
                msg = "default-fail"
                try:
                    # Check authentication
                    valid = data['sessionid'] in self.config['sessions']
                    if valid:
                        # Add client to connected clients set
                        self.orchestrator.authorized_clients.add(websocket)
                        msg = "client added"
                    else:
                        raise Exception("no sessionid provided")
                except Exception as e:
                    valid = False
                    Log.log(f"WS: Authentication failed: {e}")
                    msg = "auth-fail"

                if valid:
                    # If valid ping, return received
                    if data['category'] == 'ping':
                        msg = "authenticated"
                    else:
                        try:
                            msg = self.orchestrator.ws_command(data, websocket)
                        except Exception as e:
                            Log.log(f"WS: Failed to run ws_command: {e}")
                            valid = False
                            msg = "ws_command:operation-fail"

                res = WSUtil.make_response(data['id'], valid, msg)
                await websocket.send(res)

        except websockets.ConnectionClosed:
            Log.log("WS: Connection closed")

        finally:
            # Remove client from connected clients set
            self.orchestrator.authorized_clients.remove(websocket)

    async def broadcast_message(self):
        while True:
            for client in self.orchestrator.authorized_clients.copy():
                if client.open:
                    message = self.orchestrator.structure
                    await client.send(json.dumps(message))
                else:
                    self.orchestrator.authorized_clients.remove(client)
            await asyncio.sleep(0.5)  # Send the message twice a second

    def run(self):
        try:
            Log.log("WS: Starting WebSocket Thread")
            asyncio.set_event_loop(asyncio.new_event_loop())
            server = websockets.serve(self.handle, self.host, self.port)
            asyncio.get_event_loop().create_task(self.broadcast_message())
            asyncio.get_event_loop().run_until_complete(server)
            asyncio.get_event_loop().run_forever()
        except Exception as e:
            Log.log(f"WS: Failed to start WebSocket Thread: {e}")
