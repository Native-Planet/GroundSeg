import os
import json
import asyncio
from datetime import datetime, timedelta

class Threader:
    def __init__(self,state):
        self.state = state
        self.state['ready']['threader'] = True 

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
                        '''
                        login = self.ws_util.structure['system']['login']
                        setup = self.ws_util.structure['system']['setup']
                        '''
                        login = "1"
                        setup = "2"
                        message = {"system": {"login":login,"setup":setup}}
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
                        #message = self.state['broadcast']
                        #message['system']['login']['access'] = "authorized"
                        login = "1000"
                        setup = "2000"
                        message = {"system": {"login":login,"setup":setup}}
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

    async def urbits(self):
        while True:
            # urbits_loop
            await asyncio.sleep(1)


    async def urbits_loop(self):
        from threader.urbits import UrbitsLoop
        

