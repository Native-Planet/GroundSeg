from time import sleep

# Docker
from dockers.netdata import Netdata
from dockers.wireguard import Wireguard
from dockers.minio import MinIO
from dockers.urbit import Urbit
from dockers.webui import WebUI

# Websocket
from lib.system import WSSystem
from lib.urbits import WSUrbits
from lib.minios import WSMinIOs

# Util
from log import Log

class Orchestrator:
    def __init__(self, state):
        self.state = state
        self.structure = self.state['broadcast']
        self._debug = self.state['debug']
        self.broadcaster = self.state['broadcaster']

        # Config
        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

        # StarTram API
        self.api = None
        while self.api == None:
            try:
                self.api = self.state['startram']
            except:
                pass
            sleep(2)

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
        self.urbit = self.state['dockers']['urbit'] = Urbit(self.config_object, self.wireguard, self.minio)
        self.webui = self.state['dockers']['webui'] = WebUI(self.config_object)

        self.ws_system = self.state['ws']['system'] = WSSystem(self.state)
        self.ws_urbits = self.state['ws']['urbits'] = WSUrbits(self.state)
        self.ws_minios = self.state['ws']['minios'] = WSMinIOs(self.state)

        self.state['ready']['orchestrator'] = True # new ready notifier
        self.config_object.gs_ready = True # legacy, TODO: deprecate when no code depends on it
        print("orchestrator:__init__ Initialization completed")

    #
    # Combo functions
    #

    def startram_register(self, id):
        registered = "no"
        def broadcast(t):
            self.broadcaster.system_broadcast('system','startram','register',t)

        try:
            # register device
            broadcast("registering")
            try:
                region = self.grab_form(id,'startram','region') or "us-east"
            except:
                region = "us-east"

            print(region)
            try:
                reg_code = self.grab_form(id,'startram','key') or ""
            except:
                reg_code = ""

            if self.api.register_device(id,region):
                # update wg0.conf
                broadcast("updating")
                if self.api.retrieve_status(10):
                    conf = self.wireguard.anchor_data['conf'] # TODO: temporary
                    print(conf)
                    '''
                    if self.wireguard.update_wg_config(conf):
                        self.config['wgRegistered'] = True
                        self.config_object.save_config()

                        # start wg container
                        if self.wireguard.start():
                            broadcast("start-wg")
                            self.config['wgOn'] = True
                            self.config_object.save_config()

                            # start mc
                            broadcast("start-mc")
                            self.minio.start_mc()
                            broadcast("success")
                            registered = "yes"

                            # register services
                            piers = self.config['piers'].copy()
                            sub = self.wireguard.anchor_data['subdomains']
                            for patp in piers:
                                try:
                                    res = self.ws_util.services_exist(patp, sub)
                                    uw = res['urbit-web']
                                    ua = res['urbit-ames']
                                    m = res['minio']
                                    mc = res['minio-console']
                                    mb = res['minio-bucket']

                                    # One or more of the urbit services is not registered
                                    if not (uw and ua):
                                        Thread(target=self.startram_api.create_service(patp, 'urbit', 10))
                                    # One or more of the minio services is not registered
                                    if not (m and mc and mb):
                                        Thread(target=self.startram_api.create_service(f"s3.{patp}", 'minio', 10))
                                except Exception as e:
                                    Log.log(f"orchestrator:startram_register:{patp} failed to create service: {e}")

                            # Loop until all services are done
                            done = set()
                            while len(done) != len(piers):
                                if self.startram_api.retrieve_status(1):
                                    sub = self.wireguard.anchor_data['subdomains']
                                    for patp in piers: 
                                        res = self.ws_util.services_exist(patp, sub, True)
                                        urbit_ready = True
                                        minio_ready = True
                                        for svc in res:
                                            if res[svc] != "ok":
                                                if 'urbit' in svc:
                                                    urbit_ready = False
                                                else:
                                                    minio_ready = False
                                        if urbit_ready:
                                            self.ws_util.urbit_broadcast(patp, 'startram', 'urbit','registered')
                                        if minio_ready:
                                            self.ws_util.urbit_broadcast(patp, 'startram', 'minio','registered')

                                        if minio_ready and urbit_ready:
                                            done.add(patp)
                                sleep(5)

                            # toggle remote
                            ignored = self.ws_util.grab_form(sid, 'startram', 'ships')
                            if ignored == None:
                                ignored = []
                            for patp in piers:
                                remote = self.urbit._urbits[patp]['network'] == "wireguard"
                                if remote or (patp not in ignored):
                                    self.ws_urbits.access_toggle(patp, "remote")
                                else:
                                    self.ws_urbits.access_toggle(patp, "local")
                        else:
                            raise Exception("failed to start wireguard container")
                    else:
                        raise Exception("failed to update wg0.conf")
                    '''
                else:
                    raise Exception("failed to retrieve status")
            else:
                raise Exception("failed to register device")
        except Exception as e:
            Log.log(f"orchestrator:startram_register Error: {e}")
            broadcast(f"failure\n{e}")

        sleep(3)
        broadcast(registered)

    # modify form
    def edit_form(self, action):
        root = self.state['personal_broadcast']
        id = action.get('token').get('id')
        payload = action.get('payload')
        template = payload.get('template')
        if not root.get(id) or not isinstance(root[id], dict):
            root[id] = {}
        # template
        root = root[id]
        if not root.get(template) or not isinstance(root[template], dict):
            root[template] = {}

        # key, value
        root = root[template]
        item = payload.get('item')
        value = payload.get('value')

        if item == "ships":
            if isinstance(value, str):
                if value == "all":
                    root[item] = self.config['piers'].copy()
                elif value == "none":
                    root[item] = []
            elif not root.get(item) or not isinstance(root[item], list):
                root[item] = value
            else:
                for patp in value:
                    if patp in root[item]:
                        root[item].remove(patp)
                    else:
                        root[item].append(patp)
        else:
            root[item] = value

    # read form
    def grab_form(self, id, template, item):
        try:
            return self.forms[id][template][item]
        except:
            return None

    # delete form
    def delete_form(self, sid, template):
        try:
            self.form[sid][template] = {}
            return True
        except:
            return False

