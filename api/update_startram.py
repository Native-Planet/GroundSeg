from datetime import datetime
from time import sleep

from log import Log
from utils import Utils

class UpdateStarTram:
    def __init__(self, config, wg, ws_util): 
        self.config = config.config
        self.wg = wg
        self.ws_util = ws_util
        self.count = 0

    def run(self):
        Log.log("ws_system:update_startram Starting thread")
        while True:
            # temp
            self.ws_util.system_broadcast('system','startram',"restart","hide")
            self.ws_util.system_broadcast('system','startram',"cancel","hide")
            self.ws_util.system_broadcast('system','startram',"advanced",False)

            self._container()
            self._register()
            self._autorenew()
            self._expiry()
            self._region()
            self._regions()
            self._endpoint()
            self.count += 1
            sleep(1)

    def _container(self):
        # running  -  Wireguard container is running
        # stopped  -  Wireguard container is stopped
        status = "stopped"
        try:
            if self.config['wgRegistered']:
                if self.wg.wg_docker.is_running(self.wg.data['wireguard_name']):
                    status = "running"
        except:
            pass
        self.ws_util.system_broadcast('system','startram','container', status)

    def _register(self):
        # no            -  unregistered
        # yes           -  a command was sent
        # <reg loading> -  TODO
        # success       -  registered successfully
        # failure\n<err> -  Failure message
        try:
            reg = self.ws_util.structure['system']['startram']['register']
        except:
            reg = "no"
        if reg == "yes" or reg == "no":
            status = "no"
            if self.config['wgRegistered']:
                status = "yes"
            self.ws_util.system_broadcast('system','startram','register',status)

    def _autorenew(self):
        try:
            autorenew = self.wg.anchor_data['ongoing'] == 1
        except:
            autorenew = False
        self.ws_util.system_broadcast('system','startram','autorenew',autorenew)

    def _expiry(self):
        expiry = None
        try:
            expiry = self.wg.anchor_data['lease']
        except:
            pass
        self.ws_util.system_broadcast('system','startram','expiry',expiry)

    def _region(self):
        try:
            region = self.wg.anchor_data['region']
        except:
           region = None
        self.ws_util.system_broadcast('system','startram','region',region)

    def _regions(self):
        try:
            regions = Utils.convert_region_data(self.wg.region_data)
        except:
            regions = []
        self.ws_util.system_broadcast('system','startram','regions',regions)

    def _endpoint(self):
        try:
            endpoint = self.config['endpointUrl']
        except:
            endpoint = None
        self.ws_util.system_broadcast('system','startram','endpoint',endpoint)
