import os
import sys
import json
import asyncio
import subprocess
from time import sleep
from threading import Thread
from websockets.server import serve

class GroundSeg:
    def __init__(self,debug=False):
        # The entire state of GroundSeg
        self.state = {
                "config": None,           # System Configs
                "orchestrator": None,     # Main GroundSeg module
                "ws": {},                 # Websocket classes
                "startram": None,         # StarTram API
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

        # Setup StarTram API
        Thread(target=self.init_startram_api).start()

        # Setup Flask
        Thread(target=self.init_flask).start()

        # Start broadcaster class
        from broadcaster import Broadcaster
        self.state['broadcaster'] = Broadcaster(self.state)

        # start websocket
        asyncio.run(self.serve())

    # Load Config Class
    def init_config(self):
        from config.config import Config
        base_path = "/opt/nativeplanet/groundseg"
        self.state['config'] = Config(base_path, self.state)

    # Load Orchestrator Class
    def init_orchestrator(self):
        from orchestrator import Orchestrator
        self.state['orchestrator'] = Orchestrator(self.state)

    # StarTram API 
    def init_startram_api(self):
        from api.startram import StarTramAPI
        self.state['startram'] = StarTramAPI(self.state)

    # Start Flask in Thread
    def init_flask(self):
        cfg = self.state['config']
        while cfg == None:
            sleep(0.5)
            cfg = self.state['config']
        if cfg.device_mode == "c2c":
            print("start c2c")
        else:
            from legacy.groundseg_flask import GroundSegFlask
            GroundSegFlask(self.state).run()

    async def serve(self):
        host = self.state.get('host')
        port = self.state.get('port')
        if not self.kill_process(port):
            print(f"Port {port} taken. Exiting")
        else:
            # Websocket
            from api.websocket import WSGroundSeg
            ws = WSGroundSeg(self.state)

            # GallSeg
            from api.gallseg import GallSeg
            gs = GallSeg(self.state)

            async with serve(ws.handle, self.state.get('host'), self.state.get('port')):
                from threader.threader import Threader
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
                asyncio.get_event_loop().create_task(t.anchor_information())

                # Wireguard connection refresher
                #asyncio.get_event_loop().create_task(t.wireguard_refresher())

                # startram API
                this['startram'] = asyncio.create_task(t.startram_loop())
                
                # Urbit ships information
                this['urbits'] = asyncio.create_task(t.urbits_loop())

                # Session management
                this['login'] = asyncio.create_task(t.login_loop())
                
                # broadcast
                this['a_broadcast'] = asyncio.create_task(t.broadcast_authorized())
                this['u_broadcast'] = asyncio.create_task(t.broadcast_unauthorized())
                this['s_broadcast'] = asyncio.create_task(t.broadcast_setup())

                # sessions cleanup
                this['s_cleanup'] = asyncio.create_task(t.session_cleanup())

                # task watcher
                #asyncio.create_task(self.watch_tasks(this))

                await asyncio.Future()

    # Kill port for C2C
    def kill_process(self, port):
        process = False
        try:
            output = subprocess.check_output(["lsof", "-i", f"tcp:{port}"])
            pid = int(output.split()[10])
            process = True
        except subprocess.CalledProcessError:
            return True
        if process:
            try:
                os.kill(pid, 9)
                return True
            except OSError:
                return False

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
