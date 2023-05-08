#from datetime import datetime
from time import sleep
from threading import Thread

from log import Log
#from utils import Utils

class UrbitsLoop:
    def __init__(self, config, urb, ws_util): 
        self.config = config.config
        self.urb = urb
        self.wg = urb.wg
        self.ws_util = ws_util
        self.count = 0

        for patp in self.config['piers'].copy():
            self.ws_util.urbit_broadcast(patp, 'minio', 'link')
            self.ws_util.urbit_broadcast(patp, 'minio', 'unlink')

            self.ws_util.urbit_broadcast(patp, 'container','rebuild')
            self.ws_util.urbit_broadcast(patp, 'meld', 'urth')
            self.ws_util.urbit_broadcast(patp, 'click', 'exist', False)
            self.ws_util.urbit_broadcast(patp, 'vere', 'version')

            self.ws_util.urbit_broadcast(patp, 'startram', 'access', 'unregistered') # remote, local
            self.ws_util.urbit_broadcast(patp, 'startram', 'urbit-web', 'unregistered') # registered
            self.ws_util.urbit_broadcast(patp, 'startram', 'urbit-ames', 'unregistered') # registered
            self.ws_util.urbit_broadcast(patp, 'startram', 'minio', 'unregistered') # registered
            self.ws_util.urbit_broadcast(patp, 'startram', 'minio-console', 'unregistered') # registered
            self.ws_util.urbit_broadcast(patp, 'startram', 'minio-bucket', 'unregistered') # registered

    def run(self):
        Log.log("ws_urbits:urbits_loop Starting thread")
        while True:
            for patp in self.config['piers'].copy():
                #self._startram_urbit_web(patp)
                #self._vere_version(patp)
                Thread(target=self._vere_version, args=(patp,), daemon=True).start()
            self.count += 1
            sleep(1)

    #def _startram_urbit_web(self, patp):
        # check for all service registrations
        #self.wg.anchor_data['subdomains']..
        #self.ws_util.urbit_broadcast(patp,'startram','urbit-web','registering')

    def _vere_version(self, patp):
        if self.count == 0 or self.count % 30 == 0:
            try:
                if self.urb.urb_docker.is_running(patp):
                    res = self.urb.urb_docker.exec(patp, 'urbit --version')
                    if res:
                        res = res.output.decode("utf-8").strip().split("\n")[0]
                        self.ws_util.urbit_broadcast(patp, 'vere', 'version', str(res))
            except Exception as e:
                self.ws_util.urbit_broadcast(patp, 'vere', 'version', f'error: {e}')

    '''
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
        if type(self.wg.anchor_data) == str:
            autorenew = self.wg.anchor_data
        else:
            try:
                autorenew = self.wg.anchor_data['ongoing'] == 1
            except:
                autorenew = False
            self.ws_util.system_broadcast('system','startram','autorenew',autorenew)

    def _expiry(self):
        if type(self.wg.anchor_data) == str:
            expiry = self.wg.anchor_data
        else:
            try:
                expiry = self.wg.anchor_data['lease']
            except:
                expiry = None
        self.ws_util.system_broadcast('system','startram','expiry',expiry)

    def _region(self):
        if type(self.wg.anchor_data) == str:
            region = self.wg.anchor_data
        else:
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
    '''
