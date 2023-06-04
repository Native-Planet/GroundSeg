import os
import json
import asyncio
from datetime import datetime, timedelta

# Threads
from threader.startram import StarTramLoop
from threader.urbits import UrbitsLoop
from threader.login import LoginLoop
from threader.anchor_information import AnchorInformation

class Threader:
    def __init__(self,state):
        self.state = state
        self.state['ready']['threader'] = True 
        self.broadcaster = self.state['broadcaster']

    async def anchor_information(self):
        print("threader:anchor_information Starting")
        try:
            loop = AnchorInformation(self.state)
        except Exception as e:
            print(e)
        while True:
            loop.run()
            await asyncio.sleep(1)

    async def startram_loop(self):
        print("threader:startram_loop Starting")
        self.broadcaster.system_broadcast('system','startram',"restart","")
        self.broadcaster.system_broadcast('system','startram',"cancel","")
        try:
            loop = StarTramLoop(self.state)
        except Exception as e:
            print(e)
        while True:
            loop.run()
            await asyncio.sleep(1)

    async def urbits_loop(self):
        print("threader:urbits_loop Starting")
        try:
            loop = UrbitsLoop(self.state)
        except Exception as e:
            print(e)
        while True:
            loop.run()
            await asyncio.sleep(1)

    async def login_loop(self):
        print("threader:login_loop Starting")
        try:
            loop = LoginLoop(self.state)
        except Exception as e:
            print(e)
        while True:
            loop.run()
            await asyncio.sleep(1)

    async def watch_gallseg(self,gs):
        action_file = "/opt/nativeplanet/groundseg/action" # TEMP
        patp = "sampel-palnet"
        while True:
            try:
                if os.path.exists(action_file):
                    with open(action_file) as action:
                        act = json.loads(action.read())
                        res = await gs.handle(patp,act)
                        print(f"poke: {res}")
                    os.remove(action_file)
            except Exception as e:
                print(e)
            await asyncio.sleep(0.5)

    async def broadcast_unauthorized(self):
        while True:
            try:
                clients = self.state['clients']['unauthorized'].copy()
                for s in clients:
                    if s.open:
                        login = self.state.get('broadcast').get('system').get('login')
                        message = {"system": {"login":login}}
                        message['system']['login']['access'] = "unauthorized"
                        await s.send(json.dumps(message))
                    else:
                        self.state['clients']['unauthorized'].pop(s)
            except Exception as e:
                print(f"threader:broadcast_unauthorized Broadcast fail: {e}")

            await asyncio.sleep(0.5)  # Send the message twice a second

    async def broadcast_authorized(self):
        while True:
            try:
                clients = self.state['clients']['authorized'].copy()
                for s in clients:
                    if s.open:
                        id = clients.get(s).get('id')
                        message = self.state.get('broadcast')
                        message['system']['login']['access'] = "authorized"
                        message['forms'] = self.state.get('personal_broadcast').get(id)
                        await s.send(json.dumps(message))
                    else:
                        self.state['clients']['authorized'].pop(s)

            except Exception as e:
                print(f"threader:broadcast_authorized Broadcast fail: {e}")

            await asyncio.sleep(0.5)  # Send the message twice a second

    async def session_cleanup(self):
        while True:
            try:
                if self.state['ready']['config']:
                    # check if past 5 minutes
                    sessions = self.state['config'].config['sessions']['unauthorized'].copy()
                    for token in sessions:
                        created = sessions[token]['created']
                        expire = datetime.strptime(created, "%Y-%m-%d_%H:%M:%S") + timedelta(minutes=5)
                        now = datetime.now()
                        if now >= expire:
                            # remove from config
                            print(f"threader:session_cleanup Removing unauthorized token {token}")
                            self.state['config'].config['sessions']['unauthorized'].pop(token)
                            # close the user's connection
                            for websocket in self.state['clients']['unauthorized']:
                                if self.state['clients']['unauthorized'][websocket]['id'] == token:
                                    await websocket.close()

            except Exception as e:
                print(e)

            await asyncio.sleep(1)
