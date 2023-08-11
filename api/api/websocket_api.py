import json
import asyncio
import websockets

from api.authorization import Auth

class WS:
    def __init__(self, cfg, groundseg, broadcaster, host, port, dev):
        super().__init__()
        self.cfg = cfg
        self.app = groundseg
        self.dev = dev
        self.host = host
        self.port = port
        self.broadcaster = broadcaster

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
                    setups = ['start','profile','startram']
                    setup =  self.cfg.system.get('setup') in setups
                    # Now, we check if the user provided a token
                    auth = Auth(self.cfg)
                    auth_status, token = auth.check_token(token,websocket,setup)

                    # Next, we will decide what to do with the websocket connection
                    remove_from = "none"
                    tid = token.get('id')
                    req_type = payload.get('type')

                    # If authorized token
                    if auth_status:
                        # Special case for logout
                        if req_type == "logout":
                            remove_from, auth_status, token = auth.handle_logout(token,websocket,payload.get('action'))
                        else:
                            # Add the session to active sessions (does nothing if already added)
                            self.app.active_sessions['authorized'][websocket] = tid
                    # Not authorized
                    else:
                        # Special case for login
                        if req_type == "login":
                            remove_from, auth_status, token = auth.handle_login(token, payload.get('password'),websocket)
                            self.app.active_sessions['authorized'][websocket] = tid
                        else:
                            # Add the session to active sessions (does nothing if already added)
                            self.app.active_sessions['unauthorized'][websocket] = tid

                    # Removing from active sessions upon request
                    try:
                        if remove_from == "unauthorized":
                            self.app.active_sessions['unauthorized'].pop(websocket)
                        elif remove_from == "authorized":
                            self.app.active_sessions['authorized'].pop(websocket)
                            await websocket.close()
                    except:
                        pass

                    # And finally, we send the payload and auth result
                    # to GroundSeg for processing
                    asyncio.create_task(self.app.process(self.broadcaster, websocket, auth_status, setup, payload))
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
            except websockets.ConnectionClosed:
                print(f"websocket_api:handler:send connection closed")

    async def broadcast(self):
        while True:
            try:
                if self.app.ready:
                    if self.cfg.system.get('setup') != "complete":
                        await self.broadcaster.setup()
                    else:
                        await self.broadcaster.broadcast()
            except Exception as e:
                print(f"websocket_api:broadcast: {e}")
            await asyncio.sleep(0.25)

    # We start the websocket server, using handler() to handle requests
    async def run(self):
        server = await websockets.serve(self.handler, self.host, self.port)
        await server.wait_closed()
