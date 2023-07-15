import json
import asyncio
import websockets

from api.broadcaster import Broadcaster
from api.authorization import Auth

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
            try:
                # Receive Request
                token = None
                id = None
                request = await websocket.recv()
                message = json.loads(request)

                # Check if GroundSeg is ready to handle the request
                if self.app.ready:
                    #
                    # id      - a random id for the specific event, provided
                    #           by the user
                    # token   - consists of the token id and contents, if not
                    #           provided by the user, groundseg will create a
                    #           new unauthorized token
                    # payload - contents of the request
                    #
                    id = message.get('id')
                    token = message.get('token')
                    payload = message.get('payload')
                    # We check if id is available, if not we respond
                    # with a nack and the NO_ID error
                    if not id:
                        raise Exception("NO_ID")
                    # Check custom case for setup
                    setups = ['start','profile','startram','complete']
                    setup =  self.cfg.system.get('setup') in setups
                    # Now, we check if the user provided a token
                    auth_status, token = Auth(self.cfg).check_token(token,websocket,setup)
                    # Next, we will add the websocket connection to our active sessions
                    if auth_status:
                        tid = token.get('id')
                        self.app.active_sessions['authorized'][websocket] = tid
                    else:
                        self.app.active_sessions['unauthorized'][websocket] = tid
                    # And finally, we send the payload and auth result
                    # to GroundSeg for processing
                    asyncio.create_task(self.app.process(websocket, auth_status, payload))
                    # Everything ran without errors, return an ack
                    res = {"response":"ack","error":None}
                else:
                    raise Exception("NOT_READY")
            except Exception as e:
                res = {"response":"nack","error":str(e)}
            try:
                res['id'] = id
                res['type'] = "activity"
                if token:
                    res['token'] = token
                    await websocket.send(json.dumps(res))
            except Exception as e:
                print(f"websocket_api:handler:send Failed: {e}")

    async def broadcast(self):
        b = Broadcaster(self.cfg,self.app)
        while True:
            if self.app.ready:
                if self.cfg.system.get('setup') != "complete":
                    await b.setup()
                else:
                    await b.broadcast()
            await asyncio.sleep(0.5)

    # We start the websocket server, using handler() to handle requests
    async def run(self):
        server = await websockets.serve(self.handler, self.host, self.port)
        await server.wait_closed()

