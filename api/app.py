import sys
import json
import asyncio
from threading import Thread

# Pip Modules
from websockets.server import serve

# GroundSeg Modules
from threader.threader import Threader
from auth.auth import Auth

class GroundSeg:
    def __init__(self,debug=False):
        # The entire state of GroundSeg
        self.state = {
                "config": None,           # System Configs
                "orchestrator": None,     # Main GroundSeg module
                "threader": {},           # Coroutines
                "debug":debug,            # True if ./groundseg dev
                "ready": {                # Classes fully inited
                    "config":False,
                    "orchestrator":False,
                    "threader":False
                    },
                "host": '0.0.0.0',        # Websocket Host. Keep it at 0.0.0.0
                "port": '8000',           # Websocket Port
                "broadcast": {},          # Main broadcast from GroundSeg
                "personal_broadcast": {}, # {id:{broadcast}} additional/unique entries for a specific user session
                "tokens": {},             # Current active tokens (unused?)
                "dockers": {},            # config files of docker containers
                "clients": {              # websocket sessions
                    "authorized": {},
                    "unauthorized": {}
                    }
                }

    def run(self):
        # Setup System Config
        Thread(target=self.init_config).start()

        # Setup Orchestrator
        Thread(target=self.init_orchestrator).start()

        # start websocket
        asyncio.run(self.serve())

    def init_config(self):
        from config.config import Config
        base_path = "/opt/nativeplanet/groundseg"
        self.state['config'] = Config(base_path, self.state)

    def init_orchestrator(self):
        from orchestrator import Orchestrator
        self.state['orchestrator'] = Orchestrator(self.state)

    def make_activity(self, id, status_code, msg, token=None):
        res = {"activity":{id:{"message":msg,"status_code":status_code}}}
        if token:
            res['activity'][id]['token'] = token
        return json.dumps(res)

    async def handle(self, websocket):
        print("app:handle New Websocket Connection")
        try:
            async for message in websocket:
                ready = self.state.get('ready')
                config_ready = ready.get('config')
                orchestrator_ready = ready.get('orchestrator')

                action = json.loads(message)
                status_code = 1
                msg = "CONFIG_NOT_READY"
                token = None

                if config_ready:
                    # Verify the action
                    status_code, msg, token = Auth(self.state).verify_session(action, websocket)

                    # process action in orchestrator
                    if status_code == 0:
                        if orchestrator_ready:
                            status_code, msg, token = self.handle_request(
                                    action,
                                    websocket,
                                    status_code,
                                    msg,
                                    token
                                    )
                        else:
                            status_code = 1
                            msg = "ORCHESTRATOR_NOT_READY"

                # make and send activity
                activity = self.make_activity(action.get('id'), status_code, msg, token)
                await websocket.send(activity)

        except Exception as e:
            print(f"app:handle Error {e}")

    # receive action
    def handle_request(self, action, websocket, status_code, msg, token):
        print(f"app:handle_request id: {action['id']}")
        try:
            # Get the action category
            cat = action.get('payload').get('category')

            # Does nothing
            if cat == "token":
                pass

            # System
            elif cat == "system":
                if websocket in self.state['clients']['unauthorized']:
                    status_code, msg, token = self.system_action(action, websocket, status_code, msg)
                elif websocket in self.state['clients']['authorized']:
                    print(self.state['clients']['authorized'])

                '''
            elif cat == 'urbits':
                status_code, msg = self.orchestrator.ws_command_urbit(payload)

            elif cat == 'updates':
                status_code, msg = self.ws_command_updates(payload)

            elif cat == 'forms':
                status_code, msg = self.ws_command_forms(action)
                '''
            else:
                status_code = 1
                msg = "INVALID_CATEGORY"
                raise Exception(f"'{cat}' is not a valid category")
        except Exception as e:
            print(f"app:handle_request Error {e}")

        return status_code, msg, token

    # System
    def system_action(self, data, websocket, status_code, msg):
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
            status_code, msg, token = Auth(self.state).handle_login(data,
                                                                    websocket,
                                                                    status_code,
                                                                    msg
                                                                    )

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
        async with serve(self.handle, self.state.get('host'), self.state.get('port')):
            t = Threader(self.state)
            this = self.state['threader']

            #
            #   Before orchestrator
            #

            # C2C kill switch (if c2c)
            #asyncio.get_event_loop().create_task(t.c2c_killswitch())

            # binary updater
            #asyncio.get_event_loop().create_task(t.binary_updater())

            # Linux updater
            #asyncio.get_event_loop().create_task(t.linux_updater())

            # System monitoring
            #asyncio.get_event_loop().create_task(t.ram_monitor())
            #asyncio.get_event_loop().create_task(t.cpu_monitor())
            #asyncio.get_event_loop().create_task(t.temp_monitor())
            #asyncio.get_event_loop().create_task(t.disk_monitor())

            #
            #   After orchestrator
            #

            # docker updater
            #asyncio.get_event_loop().create_task(t.docker_updater())

            # Scheduled melds
            #asyncio.get_event_loop().create_task(t.meld_timer())

            # Anchor information
            #asyncio.get_event_loop().create_task(t.anchor_information())

            # Wireguard connection refresher
            #asyncio.get_event_loop().create_task(t.wireguard_refresher())

            # broadcast
            this['a_broadcast'] = asyncio.create_task(t.broadcast_authorized())
            this['u_broadcast'] = asyncio.create_task(t.broadcast_unauthorized())

            # sessions cleanup
            this['s_cleanup'] = asyncio.create_task(t.session_cleanup())

            # task watcher
            #asyncio.create_task(self.watch_tasks(this))

            await asyncio.Future()

# Args
dev = sys.argv[1] == "dev" if len(sys.argv) > 1 else False

# Announce
if dev:
    print("---------- Starting GroundSeg in debug mode ----------")
else:
    print("----------------- Starting GroundSeg -----------------")
    print("------------------ Urbit is love <3 ------------------")

# Start
groundseg = GroundSeg(dev)
groundseg.run()
