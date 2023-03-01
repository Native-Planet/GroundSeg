# Python 
import os
import time
from datetime import datetime

# Flask
from werkzeug.utils import secure_filename

# GroundSeg modules
from log import Log
from setup import Setup
from login import Login
from system_get import SysGet
from system_post import SysPost
from bug_report import BugReport

# Docker
from wireguard import Wireguard
from minio import MinIO
from urbit import Urbit
from webui import WebUI

class Orchestrator:

    wireguard = None

    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.wireguard = Wireguard(config)
        self.minio = MinIO(config, self.wireguard)
        self.urbit = Urbit(config, self.wireguard, self.minio)
        self.webui = WebUI(config)

        self.config_object.gs_ready = True
        Log.log("GroundSeg: Initialization completed")

    #
    #   Setup
    #

    def handle_setup(self, page, data):
        try:
            if page == "anchor":
                return Setup.handle_anchor(data, self.config_object, self.wireguard, self.urbit)

            if page == "password":
                if self.config_object.create_password(data['password']):
                    return 200

        except Exception as e:
            Log.log(f"Setup: {e}")

        return 401


    #
    #   Login
    #


    def handle_login_request(self, data):
        res = Login.handle_login(data, self.config_object)
        if res:
            return Login.make_cookie(self.config_object)
        else:
            return Login.failed()


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
                return self.urbit.create(urbit_id, data.get('data'))

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

                if data['data'] == 's3-update':
                    return self.urbit.set_minio(urbit_id)

                if data['data'] == 's3-unlink':
                    return self.urbit.unlink_minio(urbit_id)

            # MinIO requests
            if data['app'] == 'minio':
                pwd = data.get('password')
                if pwd != None:
                    return self.minio.create_minio(urbit_id, pwd, self.urbit)

                if data['data'] == 'export':
                    return self.minio.export(urbit_id)

            return 400

        except Exception as e:
            Log.log(f"Urbit: Post Request failed: {e}")

        return 400


    #
    #   Anchor Settings
    #

    # Get anchor registration information
    def get_anchor_settings(self):
        lease_end = None
        ongoing = False

        try:
            lease = self.wireguard.anchor_data['lease']
        except:
            lease = None

        try:
            ongoing = self.wireguard.anchor_data['ongoing'] == 1
        except:
            ongoing = False

        if lease != None:
            x = list(map(int,lease.split('-')))
            lease_end = datetime(x[0], x[1], x[2], 0, 0)

        return { "anchor": 
                {
                "wgReg": self.config['wgRegistered'],
                "wgRunning": self.wireguard.is_running(),
                "lease": lease_end,
                "ongoing": ongoing
                }
            }


    #
    #   System Settings
    #


    # Get all system information
    def get_system_settings(self):
        is_vm = "vm" == self.config_object.device_mode

        ver = str(self.config_object.version)
        if self.config['updateBranch'] == 'edge':
            settings['gsVersion'] = ver + '-edge'

        required = {
                "vm": is_vm,
                "updateMode": self.config['updateMode'],
                "minio": self.minio.minios_on,
                "containers" : SysGet.get_containers(),
                "sessions": len(self.config['sessions']),
                "gsVersion": ver
                }

        optional = {} 
        if not is_vm:
            optional = {
                    "ram": self.config_object._ram,
                    "cpu": self.config_object._cpu,
                    "temp": self.config_object._core_temp,
                    "disk": self.config_object._disk,
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
                        time.sleep(1)
                        return 200
            return 400

        #TODO
        '''
        # logs module
        if module == 'logs':
            if data['action'] == 'view':
                return self.get_log_lines(data['container'], data['haveLine'])

            if data['action'] == 'export':
                return '\n'.join(self.get_log_lines(data['container'], 0))
        '''

        # anchor module
        if module == 'anchor':
            if data['action'] == 'get-url':
                return self.config['endpointUrl']

            if data['action'] == 'toggle':
                if self.wireguard.is_running():
                    return self.wireguard.off(self.urbit, self.minio)
                return self.wireguard.on(self.minio)

            if data['action'] == 'restart':
                return self.wireguard.restart(self.urbit, self.minio)

            if data['action'] == 'change-url':
                return self.wireguard.change_url(data['url'], self.urbit)

            if data['action'] == 'register':
                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f"https://{endpoint}/{api_version}"

                if self.wireguard.build_anchor(url, data['key']):
                    self.config['wgRegistered'] = True
                    self.config['wgOn'] = True

                    for patp in self.config['piers']:
                        self.urbit.register_urbit(patp, url)

                    if self.config_object.save_config():
                        if self.wireguard.start():
                            return 200

            if data['action'] == 'unsubscribe':
                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f'https://{endpoint}/{api_version}'
                return self.wireguard.cancel_subscription(data['key'],url)

        return module

    def handle_upload(self, req):
        # change to temp mode (DO NOT SAVE CONFIG)
        if self.config['updateMode'] == 'auto':
            self.config['updateMode'] = 'temp'

        # Uploaded pier
        file = req.files['file']
        filename = secure_filename(file.filename)
        patp = filename.split('.')[0]

        # Create subfolder
        file_subfolder = f"{self.config_object.base_path}/uploaded/{patp}"
        os.makedirs(file_subfolder, exist_ok=True)

        fn = save_path = f"{file_subfolder}/{filename}"
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

            return "File exists, try uploading again"

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(req.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except Exception as e:
            Log.log(f"{patp}: Error writing to disk: {e}")

            if self.config['updateMode'] == 'temp':
                self.config['updateMode'] = 'auto'

            return "Can't write to disk"

        total_chunks = int(req.form['dztotalchunkcount'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if os.path.getsize(save_path) != int(req.form['dztotalfilesize']):
                Log.log(f"{patp}: File size mismatched")

                if self.config['updateMode'] == 'temp':
                    self.config['updateMode'] = 'auto'

                # size mismatch
                return "File size mismatched"
            else:
                Log.log(f"{patp}: Upload complete")
                return self.urbit.boot_existing(filename)
        else:
            # Not final chunk yet
            return 200

        if self.config['updateMode'] == 'temp':
            self.config['updateMode'] = 'auto'

        return 400
