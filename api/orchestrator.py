from time import sleep
from threading import Thread

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

# Legacy API
from legacy.system_post import SysPost
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

    def startram_stop(self):
        # mc
        Thread(target=self.minio.stop_mc).start()
        for p in self.urbit._urbits.copy():
            # minio
            Thread(target=self.ws_minios.stop,args=(p,)).start()
            # urbit
            if self.urbit._urbits[p]['network'] == 'wireguard':
                Thread(target=self.ws_urbits.access_toggle,args=(p,"local")).start()

        # wireguard
        if self.wireguard.stop():
            self.config['wgOn'] = False
            self.config_object.save_config()
            return True
        return False

    def startram_start(self):
        # wireguard
        if self.wireguard.start():
            self.config['wgOn'] = True
            self.config_object.save_config()
            # mc
            self.minio.start_mc()
            # minio
            for p in self.urbit._urbits.copy():
                Thread(target=self.ws_minios.start,args=(p,self.urbit._urbits[p])).start()
            return True
        return False

    def startram_restart(self):
        self.broadcaster.system_broadcast('system','startram','restart','initializing')
        # get list of patps in remote
        remote = set()
        for patp in self.config['piers']:
            if self.urbit._urbits[patp]['network'] == "wireguard":
                remote.add(patp)
        # restart startram
        self.broadcaster.system_broadcast('system','startram','restart','stopping')
        if self.startram_stop():
            self.broadcaster.system_broadcast('system','startram','restart','starting')
            if self.startram_start():
                # toggle remote
                for p in remote:
                    Thread(target=self.ws_urbits.access_toggle,args=(p,"remote")).start()
        self.broadcaster.system_broadcast('system','startram','restart','success')
        sleep(3)
        self.broadcaster.system_broadcast('system','startram','restart')

    def startram_change_endpoint(self,action):
        token_id = action.get('token').get('id')
        if token_id:
            # stop startram
            self.broadcaster.system_broadcast('system','startram','endpoint','stopping')
            if self.startram_stop():
                # delete services
                sub = self.wireguard.anchor_data.get('subdomains')
                if sub:
                    self.broadcaster.system_broadcast('system','startram','endpoint','rm-services')
                    for patp in self.config['piers'].copy():
                        res = self.services_exist(patp, sub)
                        if True in list(res.values()):
                            Thread(target=self.api.delete_service,
                                   args=(patp,'urbit')
                                   ).start()
                            Thread(target=self.api.delete_service,
                                   args=(f's3.{patp}','minio')
                                   ).start()
                # reset pubkey
                self.broadcaster.system_broadcast('system','startram','endpoint','reset-pubkey')
                self.config_object.reset_pubkey()
                # change endpoint
                self.broadcaster.system_broadcast('system','startram','endpoint','changing')
                self.config['endpointUrl'] = self.grab_form(token_id, 'startram', 'endpoint')
                self.config['wgRegistered'] = False
                self.config['wgOn'] = False
                self.config_object.save_config()

                # update information
                self.broadcaster.system_broadcast('system','startram','endpoint','updating')
                self.region_data = {}
                self.anchor_data = {}
                self.api.url = f"https://{self.config['endpointUrl']}/{self.config['apiVersion']}"
                self.api.get_regions()
                self.broadcaster.system_broadcast('system','startram','endpoint','success')
            sleep(3)
            self.broadcaster.system_broadcast('system','startram','endpoint','')

    def startram_cancel(self, action):
        token_id = action.get('token').get('id')
        self.broadcaster.system_broadcast('system','startram','cancel','cancelling')
        key = self.grab_form(token_id,'startram','cancel')
        if self.api.cancel_subscription(key):
            print("success")
            self.broadcaster.system_broadcast('system','startram','cancel','success')
        else:
            print("failed")
            self.broadcaster.system_broadcast('system','startram','cancel','failed')
        sleep(3)
        print("delete form")
        self.delete_form(token_id,"startram")
        print("clear")
        self.broadcaster.system_broadcast('system','startram','cancel','')

    # Move this to own file
    def startram_register(self, data):
        token_id = data.get('token').get('id')
        registered = "no"
        def broadcast(t):
            self.broadcaster.system_broadcast('system','startram','register',t)
        try:
            # register device
            broadcast("registering")

            try: # get region
                region = self.grab_form(token_id,'startram','region') or "us-east"
                if len(region) < 1:
                    raise Exception()
            except:
                region = "us-east"

            try: # get reg_code
                reg_code = self.grab_form(token_id,'startram','key') or ""
            except Exception as e:
                reg_code = ""

            if self.api.register_device(reg_code, region):
                # update wg0.conf
                broadcast("updating")
                if self.api.retrieve_status(10):
                    conf = self.wireguard.anchor_data['conf'] # TODO: temporary
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
                                    res = self.services_exist(patp, sub)
                                    uw = res['urbit-web']
                                    ua = res['urbit-ames']
                                    m = res['minio']
                                    mc = res['minio-console']
                                    mb = res['minio-bucket']

                                    # One or more of the urbit services is not registered
                                    if not (uw and ua):
                                        Thread(target=self.api.create_service(patp, 'urbit', 10))
                                    # One or more of the minio services is not registered
                                    if not (m and mc and mb):
                                        Thread(target=self.api.create_service(f"s3.{patp}", 'minio', 10))
                                except Exception as e:
                                    Log.log(f"orchestrator:startram_register:{patp} failed to create service: {e}")

                            # Loop until all services are done
                            done = set()
                            while len(done) != len(piers):
                                if self.api.retrieve_status(1):
                                    sub = self.wireguard.anchor_data['subdomains']
                                    for patp in piers: 
                                        res = self.services_exist(patp, sub, True)
                                        urbit_ready = True
                                        minio_ready = True
                                        for svc in res:
                                            if res[svc] != "ok":
                                                if 'urbit' in svc:
                                                    urbit_ready = False
                                                else:
                                                    minio_ready = False
                                        if urbit_ready:
                                            self.broadcaster.urbit_broadcast(patp, 'startram', 'urbit','registered')
                                        if minio_ready:
                                            self.broadcaster.urbit_broadcast(patp, 'startram', 'minio','registered')

                                        if minio_ready and urbit_ready:
                                            done.add(patp)
                                sleep(3)

                            # toggle remote
                            ignored = self.grab_form(token_id, 'startram', 'ships')
                            if ignored == None:
                                ignored = []
                            for patp in piers:
                                remote = self.urbit._urbits[patp]['network'] == "wireguard"
                                if remote or (patp not in ignored):
                                    self.ws_urbits.access_toggle(patp, "remote")
                                else:
                                    self.ws_urbits.access_toggle(patp, "local")
                            self.delete_form(token_id,"startram")
                        else:
                            raise Exception("failed to start wireguard container")
                    else:
                        raise Exception("failed to update wg0.conf")
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
            return self.state['personal_broadcast'][id][template][item]
        except:
            return None

    # delete form
    def delete_form(self, id, template):
        try:
            self.state['personal_broadcast'][id][template] = {}
            return True
        except:
            return False

    # check if service exists for patp
    def services_exist(self, patp, subdomains, is_registered=False):
        # Define services
        services = {
                    "urbit-web":False,
                    "urbit-ames":False,
                    "minio":False,
                    "minio-console":False,
                    "minio-bucket":False
                    }
        for ep in subdomains:
            ep_patp = ep['url'].split('.')[-3]
            ep_svc = ep['svc_type']
            if ep_patp == patp:
                for s in services.keys():
                    if ep_svc == s:
                        if is_registered:
                            services[s] = ep['status']
                        else:
                            services[s] = True
        return services


#
#   LEGACY
#

    # Get all details of Urbit ID
    def get_urbit(self, urbit_id):
        try:
            res = self.urbit.get_info(urbit_id)
        except Exception as e:
            res = "ERROR"
            Log.log(f"get_urbit {e}")
        return res

    # Handle POST request relating to Urbit ID
    def urbit_post(self ,urbit_id, data):
        try:
            # Boot new Urbit
            if data['app'] == 'boot-new':
                #TODO: move the entire endpoint to ws
                return self.urbit.create(urbit_id, data.get('key'), data.get('remote'))

            # Check if Urbit Pier exists
            if not self.urbit.urb_docker.get_container(urbit_id):
                return 400

            # Wireguard requests
            if data['app'] == 'wireguard':
                if data['data'] == 'toggle':
                    return self.urbit.toggle_network(urbit_id)

            # Urbit Pier requests
            if data['app'] == 'pier':
                if data['data'] == 'toggle':
                    return self.urbit.toggle_power(urbit_id)

                if data['data'] == '+code':
                    return self.urbit.get_code(urbit_id)

                if data['data'] == 'toggle-autostart':
                    return self.urbit.toggle_autostart(urbit_id)

                if data['data'] == 'swap-url':
                    return self.urbit.swap_url(urbit_id)

                if data['data'] == 'loom':
                    return self.urbit.set_loom(urbit_id,data['size'])

                if data['data'] == 'schedule-meld':
                    return self.urbit.schedule_meld(urbit_id, data['frequency'], data['hour'], data['minute'])

                if data['data'] == 'toggle-meld':
                    return self.urbit.toggle_meld(urbit_id)

                if data['data'] == 'do-meld':
                    return self.urbit.send_pack_meld(urbit_id)

                if data['data'] == 'delete':
                    return self.urbit.delete(urbit_id)

                if data['data'] == 'export':
                    return self.urbit.export(urbit_id)

                if data['data'] == 'devmode':
                    return self.urbit.toggle_devmode(data['on'], urbit_id)

                if data['data'] == 's3-unlink':
                    return self.urbit.unlink_minio(urbit_id)

            # Custom domain
            if data['app'] == 'cname':
                # reroute to websocket
                return self.domain_cname(urbit_id, data['data'])

            # MinIO requests
            if data['app'] == 'minio':
                pwd = data.get('password')
                if pwd is not None:
                    link = data.get('link')
                    # reroute to websocket
                    return self.minio_create(urbit_id, pwd, link)

                if data['data'] == 'export':
                    return self.minio.export(urbit_id)

            return 400

        except Exception as e:
            Log.log(f"Urbit: Post Request failed: {e}")

        return 400


    # Modify system settings
    def system_post(self, module, data, sessionid):

        # sessions module
        if module == 'session':
            return SysPost.handle_session(data, self.config_object, sessionid)

        # power module
        if module == 'power':
            return SysPost.handle_power(data)

        # binary module
        if module == 'binary':
            return SysPost.handle_binary(data)

        # network connectivity module
        if module == 'network':
            return SysPost.handle_network(data,self.config_object)

        # watchtower module
        if module == 'watchtower':
            return SysPost.handle_updater(data, self.config_object)

        # minIO module
        if module == 'minio':
            if data['action'] == 'reload':
                if self.minio.stop_all():
                    if self.minio.start_all():
                        sleep(1)
                        return 200
            return 400

        # swap module
        if module == 'swap':
            if data['action'] == 'set':
                val = data['val']
                if val != self.config['swapVal']:
                    if self.config['swapVal'] > 0:
                        if Utils.stop_swap(self.config['swapFile']):
                            Log.log(f"Swap: Removing {self.config['swapFile']}")
                            os.remove(self.config['swapFile'])

                    if val > 0:
                        if Utils.make_swap(self.config['swapFile'], val):
                            if Utils.start_swap(self.config['swapFile']):
                                self.config['swapVal'] = val
                                self.config_object.save_config()
                                return 200
                    else:
                        self.config['swapVal'] = val
                        self.config_object.save_config()
                        return 200

        # anchor module
        if module == 'anchor':
            if data['action'] == 'unsubscribe':
                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f'https://{endpoint}/{api_version}'
                return self.wireguard.cancel_subscription(data['key'],url)

        # logs module
        if module == 'logs':
            if data['action'] == 'view':
                return self.get_log_lines(data['container'], data['haveLine'])

            if data['action'] == 'export':
                return '\n'.join(self.get_log_lines(data['container'], 0))

        return module

    def get_log_lines(self, container, line):
        blob = ''

        try:
            if container == 'wireguard':
                blob = self.wireguard.logs()

            if container == 'netdata':
                blob = self.netdata.logs()

            if container == 'groundseg':
                return Log.get_log()[line:]

            if 'minio_' in container:
                blob = self.minio.minio_logs(container)

            if container in self.urbit._urbits:
                blob = self.urbit.logs(container)

            blob = blob.decode('utf-8').split('\n')[line:]

        except Exception:
            Log.log(f"Logs: Failed to get logs for {container}")

        return blob


