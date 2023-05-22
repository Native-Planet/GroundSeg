import asyncio
import json
from websockets.server import serve
from datetime import datetime, timedelta

class API:
    def __init__(self, config, ws_util, host='0.0.0.0', port=8000):
        self.authorized_clients = {}
        self.unauthorized_clients = {}
        self.config_object = config
        self.config = config.config
        self.ws_util = ws_util
        self.host = host
        self.port = port

    async def handle(self, websocket):
        try:
            async for message in websocket:
                data = json.loads(message)
                res = self.handle_request(data,websocket)
                await websocket.send(res)
        except Exception as e:
            print(e)

    def handle_request(self, data, websocket):
        status_code, msg, token = self.verify_session(data, websocket)

        if status_code != 0:
            return self.ws_util.make_activity(data['id'], status_code, msg, token)
        else:
            return self.ws_util.make_activity(data['id'], status_code, msg)
        '''
            try:
                # setup
                if msg == "SETUP":
                    return self.ws_util.make_activity(data['id'], True, "SETUP")
                if msg == "unauthorized":
                    res = {}

                payload = data['payload']

                if data['category'] == 'urbits':
                    res = self.orchestrator.ws_command_urbit(payload)

                if data['category'] == 'updates':
                    res = self.ws_command_updates(payload)

                if data['category'] == 'system':
                    res = self.ws_command_system(data)

                if data['category'] == 'forms':
                    res = self.ws_command_forms(data)

                raise Exception(f"'{data['category']}' is not a valid category")
        except Exception as e:
            raise Exception(e)

        '''

    def verify_session(self, data, websocket):
        token = data.get('token')
        cat = data['payload']['category']
        token_object = None
        try:
            if token == None:
                raise Exception()

            i = token['id']
            t = token['token']
            if self.ws_util.check_token_hash(i,t):
                d = self.ws_util.keyfile_decrypt(t,self.config['keyFile'])
                if self.ws_util.check_token_content(websocket,d):
                    if d.get('authorized'):
                        self.authorized_clients[websocket] = token
                        try:
                            self.unauthorized_clients.pop(websocket)
                        except:
                            pass
                    else:
                        self.unauthorized_clients[websocket] = token
                        try:
                            self.authorized_clients.pop(websocket)
                        except:
                            pass

                    status_code = 0
                    msg = "RECEIVED"
                else:
                    raise Exception()
            else:
                raise Exception()
        except:
            if cat != "token":
                status_code = 1
                msg = "not authorized"
            else:
                token_object = self.create_token(data,websocket)
                status_code = 2
                msg = "NEW_TOKEN"

        return status_code, msg, token_object

    def create_token(self, data, websocket):
        ip = websocket.remote_address[0]
        user_agent = websocket.request_headers.get('User-Agent')
        cat = data['payload']['category']
        if cat == "token":
            # create token
            id = self.ws_util.new_secret_string(32)
            secret = self.ws_util.new_secret_string(128)
            padding = self.ws_util.new_secret_string(32)
            now = datetime.now().strftime("%Y-%m-%d_%H:%M:%S")
            contents = {
                    "id":id,
                    "ip":ip,
                    "user_agent":user_agent,
                    "secret":secret,
                    "padding":padding,
                    "authenticated":False,
                    "created":now
                    }
            k = self.config['keyFile']
            text = self.ws_util.keyfile_encrypt(contents,k)
            self.config['sessions']['unauthorized'][id] = {
                    "hash": self.ws_util.hash_string(text),
                    "created": now
                    }
            self.config_object.save_config()
            return {
                    "token": {
                        "id":id,
                        "token":text
                        }
                    }

    async def broadcast_message(self):
        #count = 0
        while True:
            try:
                clients = self.unauthorized_clients.copy()
                for client in clients:
                    if client.open:
                        #if (count % 20) == 0:
                        #    print(client)
                        message = self.ws_util.structure.copy()
                        message = {
                                "system":{
                                    "login":"unauthenticated",
                                    "access": "allowed",
                                    "attempts": 0,
                                    "cooldown": 0
                                    },
                                "setup": {
                                    "status": "done"
                                    }
                                }
                        # only send login and setup info
                        '''
                        sid = self.authorized_clients[client]
                        try:
                            forms = self.ws_util.forms.get(sid)
                            if forms != None:
                                forms = {"forms":forms}
                            else:
                                raise Exception()
                        except:
                            forms = {}
                        message = {**forms, **message}
                        '''
                        await client.send(json.dumps(message))
                    else:
                        self.unauthorized_clients.pop(client)
            except Exception as e:
                Log.log(f"websocket_handler:broadcast_message Broadcast fail: {e}")
            '''
            try:
                clients = self.authorized_clients.copy()
                for client in clients:
                    if client.open:
                        #if (count % 20) == 0:
                        #    print(client)
                        message = self.ws_util.structure.copy()
                        sid = self.authorized_clients[client]
                        try:
                            forms = self.ws_util.forms.get(sid)
                            if forms != None:
                                forms = {"forms":forms}
                            else:
                                raise Exception()
                        except:
                            forms = {}
                        message = {**forms, **message}
                        await client.send(json.dumps(message))
                    else:
                        self.authorized_clients.pop(client)
            except Exception as e:
                Log.log(f"websocket_handler:broadcast_message Broadcast fail: {e}")
            '''
            await asyncio.sleep(0.5)  # Send the message twice a second
            #count += 1

    async def serve(self):
        async with serve(self.handle, self.host, self.port):
            await asyncio.get_event_loop().create_task(self.broadcast_message())
            await asyncio.Future()

    def run(self):
        asyncio.run(self.serve())
'''
from log import Log

class GSWebSocket:
    authorized_clients = {}
    unauthorized_clients = {} # unused for now

    def __init__(self, config, orchestrator, ws_util, host='0.0.0.0', port=8000):
        self.config = config.config
        self.orchestrator = orchestrator
        self.ws_util = ws_util
        self.host = host
        self.port = port


    async def handle(self, websocket, path):
        print(websocket)
        print(path)
        await websocket.send(json.dumps({"a":"b"}))
        try:
            async for message in websocket:
                data = json.loads(message)
                valid = True
                msg = "default-fail"
                if self.config['firstBoot']:
                    if self.setup_user == None:
                        msg = "SETUP"
                        self.setup_user = websocket
                    elif self.setup_user == websocket:
                        msg = self.orchestrator.setup_command(data)
                    else:
                        Log.log(f"websocket_handler:handle_setup setup is in progress on another device!")
                        valid = False
                        msg = "not-allowed"
                else:
                    try:
                        # Check authentication
                        sid = data.get('sessionid')
                        valid = sid in self.config['sessions']
                        if valid:
                            # Add client to whitelist
                            self.authorized_clients[websocket] = sid
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

                res = self.ws_util.make_activity(data['id'], valid, msg)
                await websocket.send(res)
        except websockets.ConnectionClosed:
            Log.log("WS: Connection closed")

        finally:
            # Remove client
            if websocket in self.authorized_clients:
                self.authorized_clients.pop(websocket)
            if websocket == self.setup_user:
                self.setup_user = None

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
        '''
