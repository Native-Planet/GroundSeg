import asyncio
import json
from threading import Thread
from websockets.server import serve
from datetime import datetime, timedelta

from log import Log

class API:
    def __init__(self, config, ws_util, orchestrator, host='0.0.0.0', port=8000):
        print("inited")
        self.config_object = config
        self.config = config.config

        self.ws_util = ws_util
        self.orchestrator = orchestrator

        self.host = host
        self.port = port

    async def handle(self, websocket):
        Log.log("websocket_handler:handle Message received")
        try:
            async for message in websocket:
                action = json.loads(message)
                activity = self.handle_request(action, websocket)
                await websocket.send(activity)
        except Exception as e:
            Log.log(f"websocket_handler:handle Error: {e}")

    def handle_request(self, data, websocket):
        Log.log(f"websocket_handler:handle_request id: {data['id']}")
        status_code, msg, token = self.verify_session(data, websocket)
        if status_code != 0:
            return self.ws_util.make_activity(data['id'], status_code, msg, token)
        try:
            cat = data.get('payload').get('category')
            if cat == "token":
                pass
            elif cat == "system":
                if websocket in self.ws_util.unauthorized_clients:
                    status_code, msg, token = self.system_action(data,websocket)

                '''
                if data['category'] == 'urbits':
                    res = self.orchestrator.ws_command_urbit(payload)

                if data['category'] == 'updates':
                    res = self.ws_command_updates(payload)

                if data['category'] == 'forms':
                    res = self.ws_command_forms(data)
                '''
            else:
                raise Exception(f"'{cat}' is not a valid category")
        except Exception as e:
            raise Exception(e)

        return self.ws_util.make_activity(data['id'], status_code, msg,token)


    def verify_session(self, data, websocket):
        token = data.get('token')
        cat = data['payload']['category']
        token_object = None

        try:
            if token == None:
                raise Exception("no token")

            i = token['id']
            t = token['token']

            if self.ws_util.check_token_hash(i,t):
                d = self.ws_util.keyfile_decrypt(t,self.config['keyFile'])
                if self.ws_util.check_token_content(websocket,d):
                    if d.get('authorized'):
                        self.ws_util.authorized_clients[websocket] = token
                        try:
                            self.ws_util.unauthorized_clients.pop(websocket)
                        except:
                            pass
                    else:
                        self.ws_util.unauthorized_clients[websocket] = token
                        try:
                            self.ws_util.authorized_clients.pop(websocket)
                        except:
                            pass

                    status_code = 0
                    msg = "RECEIVED"

                else:
                    raise Exception("incorrect contents")
            else:
                raise Exception("hash mismatch")
        except Exception as e:
            print(f"websocket_handler:verify_session {e}")

            if cat == "token":
                token_object = self.create_token(data,websocket)
                status_code = 2
                msg = "NEW_TOKEN"
            else:
                status_code = 1
                msg = "UNAUTHORIZED"

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
                    "authorized":False,
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

    def authorize_token(self, token):
        # decrypt
        k = self.config['keyFile']
        contents = self.ws_util.keyfile_decrypt(token['token'],k)

        # authorize token
        contents['authorized'] = True
        token['token'] = self.ws_util.keyfile_encrypt(contents,k)

        # get current token
        id = contents['id']
        unauth = self.config['sessions']['unauthorized'][id]

        # modify the token hash
        unauth['hash'] = self.ws_util.hash_string(token['token'])

        # move token to authorized
        self.config['sessions']['authorized'][id] = unauth

        # remove token from unauthorized
        self.config['sessions']['unauthorized'].pop(id)

        # save changes
        self.config_object.save_config()

        return {"token":token}


    # System
    def system_action(self, data, websocket):
        # hardcoded list of allowed modules
        whitelist = [
                'login',
                'startram',
                ]
        payload = data['payload']
        module = payload['module']
        action = payload['action']

        if module not in whitelist:
            raise Exception(f"{module} is not a valid module")

        if module == "login":
            if websocket in self.ws_util.unauthorized_clients:
                pwd = action.get('password')
                # check if password is correct
                if True: # self.ws_util.check_password(pwd):
                    token = self.authorize_token(data.get('token'))
                    status_code = 3
                    msg = "AUTHORIZED"

                    self.ws_util.authorized_clients[websocket] = token
                    try:
                        self.ws_util.authorized_clients.pop(websocket)
                    except:
                        pass
                else:
                    status_code = 1
                    msg = "authentication failed"

        '''
        if module == "startram":
            if action == "register":
                Thread(target=self.orchestrator.startram_register, args=(data['sessionid'],)).start()
            if action == "stop":
                Thread(target=self.orchestrator.startram_stop).start()
            if action == "start":
                Thread(target=self.orchestrator.startram_start).start()
            if action == "restart":
                Thread(target=self.orchestrator.startram_restart).start()
            if action == "endpoint":
                Thread(target=self.orchestrator.startram_change_endpoint,
                       args=(data['sessionid'],)
                       ).start()
            if action == "cancel":
                Thread(target=self.orchestrator.startram_cancel,
                       args=(data['sessionid'],)
                       ).start()

        '''
        return status_code, msg, token


    async def serve(self):
        async with serve(self.handle, self.host, self.port):
            from broadcast import Broadcast
            b = Broadcast(
                    self.ws_util.authorized_clients,
                    self.ws_util.unauthorized_clients,
                    self.ws_util
                    )
            asyncio.get_event_loop().create_task(b.authorized())
            asyncio.get_event_loop().create_task(b.unauthorized())
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
                            self.ws_util.authorized_clients[websocket] = sid
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
            if websocket in self.ws_util.authorized_clients:
                self.ws_util.authorized_clients.pop(websocket)
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
