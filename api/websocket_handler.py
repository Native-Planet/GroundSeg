import asyncio
import websockets
import json
from threading import Thread

from log import Log

class GSWebSocket(Thread):
    def __init__(self, config, orchestrator, ws_util, host='0.0.0.0', port=8000):
        super().__init__()
        self.config_class = config
        self.config = config.config
        self.orchestrator = orchestrator
        self.ws_util = ws_util
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
                            msg = self.orchestrator.ws_command(data)
                        except Exception as e:
                            Log.log(f"WS: Failed to run ws_command: {e}")
                            valid = False
                            msg = "ws_command:operation-fail"

                res = self.orchestrator.ws_util.make_activity(data['id'], valid, msg)
                await websocket.send(res)

        except websockets.ConnectionClosed:
            Log.log("WS: Connection closed")

        finally:
            # Remove client from connected clients set
            self.orchestrator.authorized_clients.remove(websocket)

    async def broadcast_message(self):
        while True:
            try:
                for client in self.orchestrator.authorized_clients.copy():
                    if client.open:
                        message = self.ws_util.structure
                        await client.send(json.dumps(message))
                    else:
                        self.orchestrator.authorized_clients.remove(client)
            except Exception as e:
                Log.log(f"WS: Broadcast fail: {e}")
            await asyncio.sleep(0.5)  # Send the message twice a second

    async def urbits_broadcast(self):
        while True:
            try:
                for patp in self.orchestrator.urbit._urbits.copy():
                    try:
                        click = self.orchestrator.urbit._urbits[patp]['click']
                    except:
                        click = False
                    self.ws_util.urbit_broadcast(patp, 'click', 'exist', click)
            except Exception as e:
                Log.log(f"WS: Urbits Broadcast fail: {e}")
            await asyncio.sleep(1)

    def run(self):
        try:
            Log.log("WS: Starting WebSocket Thread")
            asyncio.set_event_loop(asyncio.new_event_loop())
            server = websockets.serve(self.handle, self.host, self.port)
            asyncio.get_event_loop().create_task(self.urbits_broadcast())
            asyncio.get_event_loop().create_task(self.broadcast_message())
            asyncio.get_event_loop().run_until_complete(server)
            asyncio.get_event_loop().run_forever()
        except Exception as e:
            Log.log(f"WS: Failed to start WebSocket Thread: {e}")
