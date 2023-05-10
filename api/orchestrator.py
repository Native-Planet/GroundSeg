# Python 
import os
import time
import socket
import subprocess
from time import sleep
from datetime import datetime
from threading import Thread

# Flask
from werkzeug.utils import secure_filename

# GroundSeg modules
from log import Log
from setup import Setup
from login import Login
from system_get import SysGet
from system_post import SysPost
from bug_report import BugReport
from utils import Utils
from startram_api import StartramAPI

# Websocket
from ws_system import WSSystem
from ws_urbits import WSUrbits
from ws_minios import WSMinIOs

# Docker
from netdata import Netdata
from wireguard import Wireguard
from minio import MinIO
from urbit import Urbit
from webui import WebUI

class Orchestrator:

    wireguard = None
    authorized_clients = set()
    unauthorized_clients = set() # unused for now

    def __init__(self, config, ws_util, debug=False):
        self._debug = debug
        self.config_object = config
        self.config = config.config
        self.ws_util = ws_util

        if self.config['updateMode'] == 'auto':
            count = 0
            while not self.config_object.update_avail:
                count += 1
                if count >= 10:
                    break
                Log.log("Updater: Updater information not yet ready. Checking in 3 seconds")
                sleep(3)

        self.wireguard = Wireguard(config)
        self.netdata = Netdata(config)
        self.minio = MinIO(config, self.wireguard)
        self.urbit = Urbit(config, self.wireguard, self.minio)
        self.webui = WebUI(config)
        self.startram_api = StartramAPI(config, self.wireguard, self.ws_util)

        # TODO: temp
        self.ws_init(config, debug)

        self.config_object.gs_ready = True
        Log.log("GroundSeg: Initialization completed")

    #
    #   Websocket API
    #

    # Duplicate of __init__ for future use
    def ws_init(self, config, debug):
        self.ws_system = WSSystem(self.config_object, self.wireguard, self.ws_util)
        self.ws_urbits = WSUrbits(self.config_object, self.urbit, self.ws_util)
        self.ws_minios = WSMinIOs(self.minio, self.ws_util)

    # Main command function
    def ws_command(self, data):
        try:
            payload = data['payload']
            # if category is urbits
            if data['category'] == 'urbits':
                return self.ws_command_urbit(payload)

            if data['category'] == 'updates':
                return self.ws_command_updates(payload)

            if data['category'] == 'system':
                return self.ws_command_system(data)

            if data['category'] == 'forms':
                return self.ws_command_forms(data)

            raise Exception(f"'{data['category']}' is not a valid category")
        except Exception as e:
            raise Exception(e)


    #
    # Category command functions
    #

    # Forms
    def ws_command_forms(self, data):
        try:
            # hardcoded whitelist
            whitelist = [
                    'startram'
                    ]

            payload = data['payload']
            sid = data['sessionid']
            template = payload['template']

            if template in whitelist:
                if template == "startram":
                    self.ws_util.edit_form(data, template)

        except Exception as e:
            raise Exception(e)
        return "succeeded"

    # System
    def ws_command_system(self, data):
        # hardcoded list of allowed modules
        whitelist = [
                'startram',
                ]
        payload = data['payload']
        module = payload['module']
        action = payload['action']

        if module not in whitelist:
            raise Exception(f"{module} is not a valid module")

        if module == "startram":
            if action == "register":
                Thread(target=self.startram_register, args=(data['sessionid'],)).start()

        return "succeeded"

    # Urbit
    def ws_command_urbit(self, payload):
        # hardcoded list of allowed modules
        whitelist = [
                'meld',
                'minio',
                'container'
                ]
        patp = payload['patp']
        module = payload['module']
        action = payload['action']

        if module not in whitelist:
            raise Exception(f"{module} is not a valid module")

        # MinIO
        if module == "minio":
            if action == "link":
                Thread(target=self.minio_link, args=(patp,)).start()
            if action == "unlink":
                Thread(target=self.minio_unlink, args=(patp,)).start()

        # Pack and Meld
        if module == "meld":
            if action == "urth":
                Thread(target=self.ws_urbits.meld_urth,
                       args=(patp,)
                       ).start()

        # Urbit Docker Container
        if module == "container":
            if action == "rebuild":
                Thread(target=self.ws_urbits.container_rebuild,
                       args=(patp,)
                       ).start()

        return "succeeded"

    # Updates
    def ws_command_updates(self, payload):
        # hardcoded list of allowed modules
        whitelist = [
                'linux',
                ]
        module = payload['module']
        action = payload['action']

        if module not in whitelist:
            raise Exception(f"{module} is not a valid module")

        # linux updates
        if module == "linux":
            # clear update action
            if action == "refresh": 
                Thread(target=self.ws_util.system_broadcast,
                       args=('updates','linux','update','updated')
                       ).start()
            if action == "update":
                Thread(target=self.ws_system.linux_update).start()

        return "succeeded"

    #
    # Combo functions
    #

    def startram_register(self, sid):
        registered = "no"
        def broadcast(t):
            self.ws_util.system_broadcast('system','startram','register',t)

        try:
            # register device
            broadcast("registering")
            if self.startram_api.register_device(sid):
                # update wg0.conf
                broadcast("updating")
                if self.startram_api.retrieve_status(10):
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
                            sub = self.wireguard.anchor_data['subdomains']
                            for patp in self.config['piers'].copy():
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

                            Log.log("10 sec here")
                            sleep(10)

                            # Loop forever until all is done
                            for patp in self.config['piers'].copy():
                                self.ws_util.urbit_broadcast(patp, 'startram', 'minio','registered')
                                self.ws_util.urbit_broadcast(patp, 'startram', 'urbit','registered')
                                sleep(1)

                            for patp in self.config['piers'].copy():
                                self.ws_util.urbit_broadcast(patp, 'startram', 'access', 'to-remote')
                                sleep(1)

                            # toggle remote
                            for patp in self.config['piers'].copy():
                                self.ws_util.urbit_broadcast(patp, 'startram', 'access', 'remote')
                                sleep(1)
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


    def minio_link(self, patp):
        # create minio service account
        pier_config = self.urbit._urbits[patp]
        acc, secret = self.ws_minios.create_account(pier_config)
        if acc and secret:
            bucket = 'bucket'
            # set in urbit
            self.ws_urbits.minio_link(pier_config, acc, secret, bucket)
        else:
            Log.log(f"WS: {patp} minio:link failed") 

    def minio_unlink(self, patp): # temp
        pier_config = self.urbit._urbits[patp]
        self.ws_urbits.minio_link(pier_config, unlink=True) # unlink shorthand

    def minio_create(self, patp, pwd, link):
        res = self.minio.create_minio(patp, pwd, self.urbit)
        if link and (res == 200):
            self.minio_link(patp)
        return res

    def domain_cname(self, patp, data):
        link = data['relink']
        res = self.urbit.custom_domain(patp, data)
        if link and (res == 200):
            self.minio_link(patp)
        return res

    #
    #   Setup
    #

    def handle_setup(self, page, data):
        content = page
        try:
            if content == "regions":
                return Setup.handle_regions(data, self.config, self.wireguard)
            if page == "anchor":
                return Setup.handle_anchor(data, self.config_object, self.wireguard, self.urbit, self.minio)

            if page == "password":
                return Setup.handle_password(data, self.config_object)

        except Exception as e:
            Log.log(f"Setup: {e}")

        return 401


    #
    #   Login
    #


    def handle_login_request(self, data):
        now = datetime.now()
        s = self.config_object.login_status
        unlocked = s['end'] < now
        if unlocked:
            res = Login.handle_login(data, self.config_object)
            if res:
                return Login.make_cookie(self.config_object)
        return Login.failed(self.config_object, s['end'] < now)

    def handle_login_status(self):
        try:
            now = datetime.now()
            remainder = 0
            s = self.config_object.login_status
            locked = False
            if s['end'] > now:
                remainder = int((s['end'] - now).total_seconds())
                locked = s['locked']

            return {"locked": locked, "remainder": remainder}
            
        except Exception as e:
            Log.log(f"Login: Failed to get login status: {e}")
            return 400

    #
    #   Bug Report
    #


    def handle_report(self, data):
        bp = self.config_object.base_path
        return BugReport.submit_report(data, bp, self.config['wgRegistered'])


    #
    #   Urbit Pier
    #


    # List of Urbit Ships in Home Page
    def get_urbits(self):
        return self.urbit.list_ships()

    # Get all details of Urbit ID
    def get_urbit(self, urbit_id):
        return self.urbit.get_info(urbit_id)

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


    #
    #   System Settings
    #

    # Get all system information
    def get_system_settings(self):
        is_vm = "vm" == self.config_object.device_mode

        ver = str(self.config_object.version)
        if self.config['updateBranch'] != 'latest':
            ver = f"{ver}-{self.config['updateBranch']}"

        ui_branch = ""
        if self.webui.data['webui_version'] != 'latest':
            ui_branch = f"-{self.webui.data['webui_version']}"

        required = {
                "vm": is_vm,
                "updateMode": self.config['updateMode'],
                "minio": self.minio.minios_on,
                "containers" : SysGet.get_containers(),
                "sessions": len(self.config['sessions']),
                "gsVersion": ver,
                "uiBranch": ui_branch,
                "ram": self.config_object._ram,
                "cpu": self.config_object._cpu,
                "temp": self.config_object._core_temp,
                "disk": self.config_object._disk,
                "netdata": f"http://{socket.gethostname()}.local:{self.netdata.data['port']}",
                "swapVal": self.config['swapVal'],
                "maxSwap": Utils.max_swap(self.config['swapFile'], self.config['swapVal'])
                }

        optional = {} 
        if not is_vm:
            optional = {
                    "connected": SysGet.get_connection_status(),
                    "ethOnly": SysGet.get_ethernet_status()
                    }

        settings = {**optional, **required}
        return {'system': settings}

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
            if data['action'] == 'toggle':
                if self.wireguard.is_running():
                    return self.wireguard.off(self.urbit, self.minio)
                return self.wireguard.on(self.minio)

            if data['action'] == 'restart':
                return self.wireguard.restart(self.urbit, self.minio)

            if data['action'] == 'change-url':
                return self.wireguard.change_url(data['url'], self.urbit, self.minio)

            '''
            try:
                if data['action'] == 'register':
                    endpoint = self.config['endpointUrl']
                    api_version = self.config['apiVersion']
                    url = f"https://{endpoint}/{api_version}"

                    if self.wireguard.build_anchor(url, data['key'], data['region']):
                        self.minio.start_mc()
                        self.config['wgRegistered'] = True
                        self.config['wgOn'] = True

                        for patp in self.config['piers']:
                            self.urbit.register_urbit(patp, url)

                        if self.config_object.save_config():
                            if self.wireguard.start():
                                return 200
            except Exception as e:
                Log.log(f"Anchor: Failed to register endpoint")
            '''

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


    #
    #   Pier Upload
    #


    def upload_status(self, data):
        try:
            patp = data['patp']
            if data['action'] == 'status':
                try:
                    res = self.config_object.upload_status[patp]
                    if res['status'] == 'extracting':
                        res['progress']['current'] = self.get_directory_size(f"{self.config['dockerData']}/volumes/{patp}/_data")
                        return res
                    return res
                except Exception as e:
                    Log.log(f"Upload: Failed to get status {e}")
                    return {'status':'none'}

            if data['action'] == 'remove':
                self.config_object.upload_status.pop(patp)
                return {'status':'removed'}

        except Exception as e:
            Log.log(f"Upload: Failed to get upload status: {e}")
            return {'status':'none'}

    def get_directory_size(self, directory):
        total_size = 0
        with os.scandir(directory) as it:
            for entry in it:
                if entry.is_file():
                    total_size += entry.stat().st_size
                elif entry.is_dir():
                    total_size += self.get_directory_size(entry.path)
        return total_size



    def handle_upload(self, req):
        # change to temp mode (DO NOT SAVE CONFIG)
        if self.config['updateMode'] == 'auto':
            self.config['updateMode'] = 'temp'

        # Uploaded pier
        remote = False
        try:
            for f in req.files:
                con = f
                break

            remote = False
            fix = False

            if 'remote' in con:
                remote = True
            if 'yes' in con:
                fix = True
            file = req.files[con]

        except Exception as e:
            Log.log(f"Upload: File request fail: {e}")
            return "Invalid file type"

        filename = secure_filename(file.filename)
        patp = filename.split('.')[0]

        self.config_object.upload_status[patp] = {'status':'uploading'}

        # Create subfolder
        file_subfolder = f"{self.config_object.base_path}/uploaded/{patp}"
        os.makedirs(file_subfolder, exist_ok=True)

        save_path = f"{file_subfolder}/{filename}"
        current_chunk = int(req.form['dzchunkindex'])

        if current_chunk == 0:
            try:
                Log.log(f"{patp}: Starting upload")
                os.remove(save_path)
                Log.log(f"{patp}: Cleaning up old files")
            except:
                Log.log(f"{patp}: Directory is clear")

        if os.path.exists(save_path) and current_chunk == 0:
            os.remove(save_path)

            if self.config['updateMode'] == 'temp':
                self.config['updateMode'] = 'auto'
                self.config_object.save_config()

            return "File exists, try uploading again"

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(req.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except Exception as e:
            Log.log(f"{patp}: Error writing to disk: {e}")

            if self.config['updateMode'] == 'temp':
                self.config['updateMode'] = 'auto'
                self.config_object.save_config()

            return "Can't write to disk"

        total_chunks = int(req.form['dztotalchunkcount'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if os.path.getsize(save_path) != int(req.form['dztotalfilesize']):
                Log.log(f"{patp}: File size mismatched")

                if self.config['updateMode'] == 'temp':
                    self.config['updateMode'] = 'auto'
                    self.config_object.save_config()

                # size mismatch
                return "File size mismatched"
            else:
                Log.log(f"{patp}: Upload complete")
                #TODO: move the entire endpoint to ws
                res = self.urbit.boot_existing(filename, remote, fix)
                if self.config['updateMode'] == 'temp':
                    self.config['updateMode'] = 'auto'
                    self.config_object.save_config()
                return res

        else:
            # Not final chunk yet
            return 200

        if self.config['updateMode'] == 'temp':
            self.config['updateMode'] = 'auto'
            self.config_object.save_config()

        return 400
