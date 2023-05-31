import sys
import json
import asyncio
from threading import Thread

# Pip Modules
from websockets.server import serve

# GroundSeg Modules
from threader.threader import Threader
from broadcaster import Broadcaster

class GroundSeg:
    def __init__(self,debug=False):
        # The entire state of GroundSeg
        self.state = {
                "config": None,           # System Configs
                "orchestrator": None,     # Main GroundSeg module
                "threader": {},           # Coroutines
                "broadcaster": None,      # Broadcaster util class
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

        # Start broadcaster class
        self.state['broadcaster'] = Broadcaster(self.state)
        # start websocket
        asyncio.run(self.serve())

    def init_config(self):
        from config.config import Config
        base_path = "/opt/nativeplanet/groundseg"
        self.state['config'] = Config(base_path, self.state)

    def init_orchestrator(self):
        from orchestrator import Orchestrator
        self.state['orchestrator'] = Orchestrator(self.state)

    async def serve(self):
        # Websocket
        from api.websocket import WSGroundSeg
        ws = WSGroundSeg(self.state)

        # GallSeg
        from api.gallseg import GallSeg
        gs = GallSeg(self.state)

        async with serve(ws.handle, self.state.get('host'), self.state.get('port')):
            t = Threader(self.state)
            this = self.state['threader']
            # Start GallSeg API
            this['gallseg'] = asyncio.create_task(t.watch_gallseg(gs))

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

            # Websocket Classes
            '''
            this['system'] =  System(self.state)
            this['urbits'] = Urbits(self.state)
            this['minios'] = MinIOs(self.state)
            '''
            
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
