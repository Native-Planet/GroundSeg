from time import sleep

from config.utils import Utils

class StarTramLoop:
    def __init__(self, state): 
        self.state = state
        self.broadcaster = self.state['broadcaster']

        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

        self.wg = None
        while self.wg == None:
            try:
                self.wg = self.state['dockers']['wireguard']
            except:
                sleep(0.5)

    def run(self):
        self._container()
        self._register()
        self._autorenew()
        self._expiry()
        self._region()
        self._regions()
        self._endpoint()

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
        self.broadcaster.system_broadcast('system','startram','container', status)

    def _register(self):
        # no             -  unregistered
        # yes            -  registered
        # registering    -  attempting /register
        # updating       -  updating wg0.conf
        # start-wg       -  starting wireguard container
        # start-mc       -  starting mc_docker
        # success        -  registered successfully
        # failure\n<err> -  Failure message
        try:
            reg = self.state['broadcast']['system']['startram']['register']
        except:
            reg = "no"

        if reg == "yes" or reg == "no":
            status = "no"
            if self.config['wgRegistered']:
                status = "yes"
            self.broadcaster.system_broadcast('system','startram','register',status)

    def _autorenew(self):
        if type(self.wg.anchor_data) == str:
            autorenew = self.wg.anchor_data
        else:
            try:
                autorenew = self.wg.anchor_data['ongoing'] == 1
            except:
                autorenew = False
            self.broadcaster.system_broadcast('system','startram','autorenew',autorenew)

    def _expiry(self):
        if type(self.wg.anchor_data) == str:
            expiry = self.wg.anchor_data
        else:
            try:
                expiry = self.wg.anchor_data['lease']
            except:
                expiry = None
        self.broadcaster.system_broadcast('system','startram','expiry',expiry)

    def _region(self):
        if type(self.wg.anchor_data) == str:
            region = self.wg.anchor_data
        else:
            try:
                region = self.wg.anchor_data['region']
            except:
               region = None
        self.broadcaster.system_broadcast('system','startram','region',region)

    def _regions(self):
        try:
            regions = Utils.convert_region_data(self.wg.region_data)
        except:
            regions = []
        self.broadcaster.system_broadcast('system','startram','regions',regions)

    def _endpoint(self):
        try:
            busy= ['stopping','rm-services','reset-pubkey','changing','updating','success']
            ep = self.broadcaster.structure.get('system', {}
                                              ).get('startram', {}
                                                    ).get('endpoint', "")
            # update information
            if ep not in busy:
                endpoint = self.config['endpointUrl']
            else:
                return
        except:
            endpoint = None
        self.broadcaster.system_broadcast('system','startram','endpoint',endpoint)
