from time import sleep

# Docker
from dockers.netdata import Netdata
from dockers.wireguard import Wireguard
from dockers.minio import MinIO
from dockers.urbit import Urbit
from dockers.webui import WebUI

# Websocket
'''
from lib.system import WSSystem
from lib.urbits import WSUrbits
from lib.minios import WSMinIOs
'''

class Orchestrator:
    def __init__(self, state):
        self.state = state
        self.config_object = self.state['config']
        self.structure = self.state['broadcast']
        self._debug = self.state['debug']

        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']

        self.config = self.config_object.config

        '''
        if self.config['updateMode'] == 'auto':
            count = 0
            while not self.config_object.update_avail:
                count += 1
                if count >= 10:
                    break
                print("orchestrator:__init__ Updater information not yet ready. Checking in 3 seconds")
                sleep(3)
        '''

        self.wireguard = self.state['dockers']['wireguard'] = Wireguard(self.config_object)
        self.netdata = self.state['dockers']['netdata'] = Netdata(self.config_object)
        self.minio = self.state['dockers']['minio'] = MinIO(self.config_object, self.wireguard)
        self.urbit = self.state['dockers']['wireguard'] = Urbit(self.config_object, self.wireguard, self.minio)
        self.webui = self.state['dockers']['webui'] = WebUI(self.config_object)
        # self.startram_api = StartramAPI(self.config, self.wireguard, self.ws_util)

        self.state['ready']['orchestrator'] = True # new ready notifier
        self.config_object.gs_ready = True # legacy, TODO: deprecate when no code depends on it
        print("orchestrator:__init__ Initialization completed")
