import json
import asyncio

class Threader:
    def __init__(self,state):
        self.state = state
        self.state['ready']['threader'] = True 

    async def broadcast_unauthorized(self):
        count = 0
        while True:
            #self.state.get('broadcast')
            for websocket in self.state.get('clients').get('unauthorized'):
                try:
                    await websocket.send(json.dumps({"hey":count}))
                except Exception as e:
                    print(e)
                    self.state['clients']['unauthorized'].pop(websocket)
                    await websocket.close()
            count += 1
            await asyncio.sleep(0.5)

    async def broadcast_authorized(self):
        while True:
            for websocket in self.state.get('clients').get('authorized'):
                await websocket.send(json.dumps({"hey":"logged in bitch"}))
            await asyncio.sleep(0.5)
