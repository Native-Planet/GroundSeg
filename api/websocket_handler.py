import asyncio
import websockets
import json
import time
import threading

from log import Log

class GSWebSocket(threading.Thread):
    def __init__(self, host='0.0.0.0', port=8000):
        super().__init__()
        self.host = host
        self.port = port
        self.connected_clients = set()

    async def handle(self, websocket, path):
        data = await websocket.recv()
        cmd = json.loads(data)

        self.connected_clients.add(websocket)
        print(self.connected_clients)
        async for message in websocket:
            Log.log(f"WS: Request: {message}")
            await websocket.send(message)

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

    async def broadcast_message(self):
        count = 0
        while True:
            for client in self.connected_clients.copy():
                Log.log(f"broadcast message: {client}")
                if client.open:
                    message = str(count)
                    await client.send(message)
                else:
                    self.connected_clients.remove(client)
            await asyncio.sleep(1)  # Send the message every 5 seconds
            count += 1

    '''
    connected_clients = set()
    async def websocket_handler(self, websocket, path):
        try:
            # Authenticate client
            auth_message = await websocket.recv()
            auth_data = json.loads(auth_message)

            # Check authentication
            if auth_data["username"] == "user" and auth_data["password"] == "password":
                await websocket.send("Authentication successful")
                print(f"Authenticated: {auth_data['username']}")

                # Add client to connected clients set
                self.connected_clients.add(websocket)

                # Handle messages
                while True:
                    message = await websocket.recv()
                    print(f"Received message: {message}")
                    await websocket.send(f"Server response: {message}")
            else:
                await websocket.send("Authentication failed")
                print("Authentication failed")

        except websockets.ConnectionClosed:
            print("Connection closed")
        finally:
            # Remove client from connected clients set
            connected_clients.remove(websocket)

    async def broadcast_message(self):
        count = 0
        while True:
            for client in connected_clients.copy():
                if client.open:
                    message = str(count)
                    await client.send(message)
                else:
                    connected_clients.remove(client)
            await asyncio.sleep(5)  # Send the message every 5 seconds
            count += 1


    def run(self):
        start_server = websockets.serve(self.websocket_handler, "localhost", 8000)
        asyncio.get_event_loop().create_task(self.broadcast_message())
        asyncio.get_event_loop().run_until_complete(start_server)
        asyncio.get_event_loop().run_forever()
    '''
