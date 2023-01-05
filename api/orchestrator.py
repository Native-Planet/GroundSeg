import json
import requests
import os
import sys
import time
import psutil
import shutil
import copy
import subprocess
import threading
import zipfile
import tarfile
import secrets
import string
import hashlib
import socket
import base64
import urllib.request
import ssl
import nmcli

from flask import jsonify, send_file, current_app
from datetime import datetime
from io import BytesIO
from pywgkey.key import WgKey

from utils import Log, Network
from wireguard import Wireguard
from webui_docker import WebUIDocker
from urbit_docker import UrbitDocker, default_pier_config
from minio_docker import MinIODocker
from mc_docker import MCDocker

class Orchestrator:
    
#
#   Variables
#
    # System
    _ram = None
    _cpu = None
    _core_temp = None
    _disk = None

    # GroundSeg
    gs_version = 'v1.0.3-edge'
    _vm = False
    _npbox = False
    _c2c_mode = False
    _wifi_device = None
    anchor_config = {'lease': None,'ongoing': None}
    minIO_on = False
    config = {}

    # Docker
    _urbits = {}
    _minios = {}
    _mc = {}
    _webui = {}


#
#   init
#
    def __init__(self, config_file):

        # store config file location
        self.config_file = config_file

        # load existing or create new system.json
        self.config = self.load_config(config_file)
        Log.log_groundseg("Loaded system JSON")

        # if first boot, set up keys
        if self.config['firstBoot']:
            Log.log_groundseg("GroundSeg is in setup mode")
            self.reset_pubkey()
        
        # save the latest config to file
        self.save_config()

        if os.path.isfile(f"{self.config['CFG_DIR']}/vm") or 'WSL_DISTRO_NAME' in os.environ:
            Log.log_groundseg("VM mode detected. Enabling limited features")
            self._vm = True
            
        if os.path.isfile(f"{self.config['CFG_DIR']}/nativeplanet"):
            Log.log_groundseg("NP box detected. Enabling NP box features")
            self._npbox = True

        # Remove MC Binary
        if os.path.isfile(f"{self.config['CFG_DIR']}/mc"):
            Log.log_groundseg("Old MC binary found. Deleting...")
            os.system(f"rm {self.config['CFG_DIR']}/mc")
            Log.log_groundseg("Old MC binary deleted")

        # Check for internet access
        internet = self.check_internet_access()

        # NP box
        if self._npbox:
            if not internet:
                Log.log_groundseg("No internet access, enabling Connect2Connect mode")
                self._c2c_mode = True

                # Get wifi device
                for d in nmcli.device():
                    if d.device_type == 'wifi':
                        self._wifi_device = d.device
                        break

        # Not NP box
        else:
            while not internet:
                Log.log_groundseg("No internet access, checking again in 15 seconds")
                time.sleep(15)
                internet = self.check_internet_access()

        # Skip if in c2c mode
        if not self._c2c_mode:
            # start wireguard if anchor is registered
            self.wireguard = Wireguard(self.config)
            self.wireguard.stop()
            if self.config['wgRegistered'] and self.config['wgOn']:
                self.wireguard.start()
                self.toggle_minios_off()
                self.toggle_minios_on()
                Log.log_groundseg("Wireguard connection started")

            # load urbit ships
            self.load_urbits()

            # Create password if doesn't exist
            if len(self.config['pwHash']) < 1:
                self.create_password('nativeplanet')

            # Start WebUI
            self.start_webui()

        # End of Init
        Log.log_groundseg("Initialization completed")


    # Check if device has internet access
    def check_internet_access(self):
        try:
            context = ssl._create_unverified_context()
            urllib.request.urlopen('https://nativeplanet.io',
                                   timeout=1, context=context
                                   )
            return True
        except Exception as e:
            Log.log_groundseg(e)
            return False

    # Checks if system.json and all its fields exists, adds field if incomplete
    def load_config(self, config_file):
        # Make config directories
        cfg_path = '/opt/nativeplanet/groundseg'
        os.makedirs(f"{cfg_path}/settings/pier", exist_ok=True)

        # Populate config
        cfg = {}
        try:
            with open(config_file) as f:
                cfg = json.load(f)
        except Exception as e:
            Log.log_groundseg(e)

        update_mode = 'latest'
        try:
            if cfg['updateUrl'] == 'https://version.infra.native.computer/version_edge.csv':
                update_mode = 'edge'
        except:
            pass

        cfg = self.check_config_field(cfg,'firstBoot',True)
        cfg = self.check_config_field(cfg,'piers',[])
        cfg = self.check_config_field(cfg,'endpointUrl', 'api.startram.io')
        cfg = self.check_config_field(cfg,'apiVersion', 'v1')
        cfg = self.check_config_field(cfg,'wgRegistered', False)
        cfg = self.check_config_field(cfg, 'wgOn', False)
        cfg = self.check_config_field(cfg,'updateMode','auto')
        cfg = self.check_config_field(cfg, 'sessions', [])
        cfg = self.check_config_field(cfg, 'pwHash', '')
        cfg = self.check_config_field(cfg, 'webuiPort', '80')
        cfg = self.check_config_field(cfg, 'updateBranch', update_mode)
        cfg = self.check_config_field(cfg, 'updateInterval', 90)
        cfg = self.check_config_field(cfg, 'c2cInterval', 0)

        cfg['gsVersion'] = self.gs_version
        cfg['CFG_DIR'] = cfg_path

        bin_hash = 'no-binary-detected'

        try:
            bin_hash = self.make_hash("/opt/nativeplanet/groundseg/groundseg")
            Log.log_groundseg(f"Binary hash: {bin_hash}")
        except:
            Log.log_groundseg("No binary detected!")

        cfg['binHash'] = bin_hash

        # Remove reg_key from old configs
        if 'reg_key' in cfg:
            if cfg['reg_key'] != None:
                cfg['wgRegistered'] = True
            cfg.pop('reg_key')

        if 'autostart' in cfg:
            cfg.pop('autostart')
        
        if 'updateUrl' in cfg:
            cfg.pop('updateUrl')

        return cfg

    def start_webui(self):
        self._webui = WebUIDocker(self.config['webuiPort'], self.config['updateBranch'])
        Log.log_groundseg("WebUI started")


    def make_hash(self, file):
        h  = hashlib.sha256()
        b  = bytearray(128*1024)
        mv = memoryview(b)
        with open(file, 'rb', buffering=0) as f:
            while n := f.readinto(mv):
                h.update(mv[:n])
        return h.hexdigest()

    # Adds missing field to config
    def check_config_field(self, cfg, field, default):
        if not field in cfg:
            cfg[field] = default
            Log.log_groundseg(f"{field} doesn't exist! Creating with default value: {default}")
        return cfg

    # Load urbit ships
    def load_urbits(self):
        if self.wireguard.wg_docker.is_running():
            Log.log_groundseg("Starting MC")
            self._mc = MCDocker()
            time.sleep(3)

        for p in self.config['piers']:

            Log.log_groundseg(f"{p}: Loading Pier")

            data = None
            with open(f'/opt/nativeplanet/groundseg/settings/pier/{p}.json') as f:
                data = json.load(f)

            # Add all missing fields
            data = {**default_pier_config, **data}

            self._urbits[p] = UrbitDocker(data)

            if data['minio_password'] != '' and self.wireguard.wg_docker.is_running():
                self._minios[p] = MinIODocker(data)
                if self.toggle_minios_on():
                    x = self._mc.mc_setup(p, self._minios[p].config['wg_s3_port'], self._minios[p].config['minio_password'])

            if self._urbits[p].config['boot_status'] == 'boot' and not self._urbits[p].running:
                self._urbits[p].start()

        Log.log_groundseg("Urbit Piers loaded")

    # Reload minios
    def reload_minios(self):
        if self.wireguard.wg_docker.is_running():
            Log.log_groundseg("Starting MC")
            self._mc = MCDocker()
            time.sleep(3)

        for p in self.config['piers']:

            data = None
            with open(f'/opt/nativeplanet/groundseg/settings/pier/{p}.json') as f:
                data = json.load(f)

            if data['minio_password'] != '' and self.wireguard.wg_docker.is_running():
                self._minios[p] = MinIODocker(data)
                if self.toggle_minios_on():
                    x = self._mc.mc_setup(p, self._minios[p].config['wg_s3_port'], self._minios[p].config['minio_password'])

            Log.log_groundseg(f"{p}: MinIO reloaded")

#
#   Setup
#

    def handle_setup(self, page, data):
        try:
            if page == "anchor":
                # set endpoint
                if 'skip' in data:
                    self.config['firstBoot'] = False
                    self.save_config()
                    return 200

                changed = self.change_wireguard_url(data['endpoint'])

                # register key
                if changed == 200:
                    registered = self.register_device(data['key'])
                    
                    if registered == 400:
                        return 401

                    if registered == 200:
                        self.config['firstBoot'] = False
                        self.save_config()

                    return registered

            if page == "password":
                self.create_password(data['password'])
                return 200

        except Exception as e:
            Log.log_groundseg(e)
            pass

        return 401


#
#   Login
#

    def handle_login_request(self, data):
        if 'password' in data:
            encoded_str = (self.config['salt'] + data['password']).encode('utf-8')
            this_hash = hashlib.sha512(encoded_str).hexdigest()

            if this_hash == self.config['pwHash']:
                Log.log_groundseg("Login: Password is correct!")
                return 200

        return 400

    def make_cookie(self):
        secret = ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(64))

        self.config['sessions'].append(secret)
        self.save_config()
        Log.log_groundseg("Created new Session ID")
        Log.log_groundseg(f"Active Sessions {len(self.config['sessions'])}")

        return secret

#
#   Bug Report
#

    def handle_bug_report(self, data):
        try:
            # Prep
            report = datetime.now().strftime('%Y-%m-%d-%H-%M-%S')
            os.system(f"mkdir -p {self.config['CFG_DIR']}/bug-reports/{report}")
            current_logfile = f"{datetime.now().strftime('%Y-%m')}.log"

            with open(f"{self.config['CFG_DIR']}/bug-reports/{report}/details.txt", "w") as f:
                f.write(f"Contact:\n{data['person']}\nDetails:\n{data['message']}")
                f.close()

            # Create zipfile
            bug_file = zipfile.ZipFile(
                    f"{self.config['CFG_DIR']}/bug-reports/{report}/{report}.zip", 'w', zipfile.ZIP_DEFLATED
                    )

            # wireguard config
            if self.config['wgRegistered']:
                try:
                    bug_file.writestr('wireguard.log', subprocess.check_output(['docker', 'logs', 'wireguard']))
                    bug_file.writestr('wg_show.txt', subprocess.check_output(
                        ['docker', 'exec', 'wireguard', 'wg', 'show']
                        ))
                except:
                    pass

            # docker ps -a
            bug_file.writestr('docker.txt', subprocess.check_output(['docker', 'ps', '-a']).decode('utf-8'))

            # current log
            bug_file.write(f"{self.config['CFG_DIR']}/logs/{current_logfile}", arcname=current_logfile)

            # save zipfile
            bug_file.close()

            # send to endpoint
            bug_endpoint = "https://bugs.groundseg.app"


            uploaded_file = open(f"{self.config['CFG_DIR']}/bug-reports/{report}/{report}.zip", 'rb')

            form_data = {"contact": data['person'], "string": data['message']}
            form_file = {"zip_file": (f"{report}.zip", uploaded_file)}

            r = requests.post(bug_endpoint, data=form_data, files=form_file)
            Log.log_groundseg(f"Bug report sent on {report}")

            return r.status_code

        except Exception as e:
            Log.log_groundseg(e)
            pass

        return 400
#
#   Urbit Pier
#
    # Get all piers for home page
    def get_urbits(self):
        urbits= []

        for urbit in self._urbits.values():
            u = dict()
            u['name'] = urbit.pier_name
            u['running'] = urbit.is_running()
            u['url'] = f'http://{socket.gethostname()}.local:{urbit.config["http_port"]}'

            if(urbit.config['network']=='wireguard'):
                u['url'] = f"https://{urbit.config['wg_url']}"

            urbits.append(u)

        return {'urbits': urbits}

    # Get all details of Urbit ID
    def get_urbit(self, urbit_id):

        # Check if Urbit Pier exists
        urb = self._urbits.get(urbit_id)
        if(urb == None):
            return 400

        # Create query result
        u = dict()

        u['name'] = urb.pier_name
        u['running'] = urb.is_running()

        u['wgReg'] = self.config['wgRegistered']
        u['wgRunning'] = self.wireguard.is_running()
        u['autostart'] = urb.config['boot_status'] != 'off'

        u['meldOn'] = urb.config['meld_schedule']
        u['timeNow'] = datetime.utcnow()
        u['frequency'] = urb.config['meld_frequency']
        u['meldLast'] = datetime.fromtimestamp(int(urb.config['meld_last']))
        u['meldNext'] = datetime.fromtimestamp(int(urb.config['meld_next']))

        hour, minute = urb.config['meld_time'][0:2], urb.config['meld_time'][2:]

        u['containers'] = self.get_pier_containers(urbit_id)
        u['meldHour'] = int(hour)
        u['meldMinute'] = int(minute)
        u['remote'] = False
        u['urbitUrl'] = f'http://{socket.gethostname()}.local:{urb.config["http_port"]}'
        u['minIOUrl'] = ""
        u['minIOReg'] = True
        u['hasBucket'] = False
        u['loomSize'] = urb.config['loom_size']

        if(urb.config['network'] == 'wireguard'):
            u['remote'] = True
            u['urbitUrl'] = f"https://{urb.config['wg_url']}"

        if self.config['wgRegistered']:
            u['minIOUrl'] = f"https://console.s3.{urb.config['wg_url']}"

        if urb.config['minio_password'] == '':
             u['minIOReg'] = False

        if urbit_id in self._minios:
            u['hasBucket'] = True

        return u

    # Handle POST request relating to Urbit ID
    def handle_urbit_post_request(self ,urbit_id, data):

        try:
            # Boot new Urbit
            if data['app'] == 'boot-new':
               x = self.boot_new_urbit(urbit_id, data.get('data'))
               if x == 0:
                 return 200

            # Check if Urbit Pier exists
            urb = self._urbits.get(urbit_id)
            if urb == None:
                return 400

            # Urbit Pier requests
            if data['app'] == 'pier':
                if data['data'] == 'toggle':
                    return self.toggle_pier_power(urb)

                if data['data'] == '+code':
                    return self.get_urbit_code(urbit_id, urb)

                if data['data'] == 's3-update':
                    return self.set_minio_endpoint(urbit_id)

                if data['data'] == 's3-unlink':
                    lens_port = self.get_urbit_loopback_addr(urbit_id)
                    return urb.unlink_minio_endpoint(lens_port)

                if data['data'] == 'schedule-meld':
                    return urb.set_meld_schedule(data['frequency'], data['hour'], data['minute'])

                if data['data'] == 'toggle-meld':
                    x = self.get_urbit_loopback_addr(urb.config['pier_name'])
                    return urb.toggle_meld_status(x)

                if data['data'] == 'do-meld':
                    lens_addr = self.get_urbit_loopback_addr(urbit_id)
                    return urb.send_meld(lens_addr)

                if data['data'] == 'export':
                    return self.export_urbit(urb)

                if data['data'] == 'delete':
                    return self.delete_urbit(urbit_id)

                if data['data'] == 'toggle-autostart':
                    return self.toggle_autostart(urbit_id)

                if data['data'] == 'loom':
                    return urb.set_loom_size(data['size'])

            # Wireguard requests
            if data['app'] == 'wireguard':
                if data['data'] == 'toggle':
                    return self.toggle_pier_network(urb)

            # MinIO requests
            if data['app'] == 'minio':
                pwd = data.get('password')
                if pwd != None:
                    return self.create_minio_admin_account(urbit_id, pwd)

                if data['data'] == 'export':
                    return self.export_minio_bucket(urbit_id)

            return 400

        except Exception as e:
            Log.log_groundseg(e)
            return 400

    # Toggle Autostart
    def toggle_autostart(self, patp):
        if self._urbits[patp].config['boot_status'] == 'off':
            if self._urbits[patp].is_running():
                self._urbits[patp].config['boot_status'] = 'boot'
            else:
                self._urbits[patp].config['boot_status'] = 'noboot'
        else:
            self._urbits[patp].config['boot_status'] = 'off'

        self._urbits[patp].save_config()
        Log.log_groundseg(f"{patp}: Boot status set to {self._urbits[patp].config['boot_status']}")

        return 200


    # Delete Urbit Pier and MiniO
    def delete_urbit(self, patp):
        urb = self._urbits[patp]
        urb.remove_urbit()
        urb = self._urbits.pop(patp)
        
        time.sleep(2)

        if(patp in self._minios.keys()):
           minio = self._minios[patp]
           minio.remove_minio()
           minio = self._minios.pop(patp)

        self.config['piers'] = [i for i in self.config['piers'] if i != patp]
        os.remove(f"/opt/nativeplanet/groundseg/settings/pier/{patp}.json")
        self.save_config()

        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f'https://{endpoint}/{api_version}'

        if self.config['wgRegistered']:
            self.wireguard.delete_service(f'{patp}','urbit',url)
            self.wireguard.delete_service(f's3.{patp}','minio',url)

        Log.log_groundseg(f"{patp}: Pier and services deleted")
        return 200

    # Export Urbit Pier
    def export_urbit(self, urb):
        if urb.is_running():
            Log.log_groundseg(f"{urb.pier_name}: Pier export requested")
            Log.log_groundseg(f"{urb.pier_name}: Stopping!")
            urb.stop()

        file_name = f'{urb.pier_name}.zip'
        memory_file = BytesIO()
        file_path=f'{urb._volume_directory}/{urb.pier_name}/_data/'

        Log.log_groundseg(f"{urb.pier_name}: Compressing pier..")

        with zipfile.ZipFile(memory_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(file_path):
                arc_dir = root[root.find('_data/')+6:]
                for file in files:
                    zipf.write(os.path.join(root, file), arcname=os.path.join(arc_dir,file))

        memory_file.seek(0)

        Log.log_groundseg(f"{urb.pier_name}: Exported!")
        return send_file(memory_file, download_name=file_name, as_attachment=True)

    # Get list of containers related to this patp
    def get_pier_containers(self, patp):
        containers = [patp]
        if patp in list(self._minios.keys()):
            containers.append(f'minio_{patp}')

        return containers

    # Toggle Pier on or off
    def toggle_pier_power(self, urb):

        if urb.is_running() == True:
            x = urb.stop()
            if x == 0:
                if urb.config['boot_status'] != 'off':
                    urb.config['boot_status'] = 'noboot'
                    urb.save_config()
                Log.log_groundseg(f"{urb.pier_name}: Stopping!")
                return 200
            return 400
        else:
            x = urb.start()
            if x == 0:
                if urb.config['boot_status'] != 'off':
                    urb.config['boot_status'] = 'boot'
                    urb.save_config()
                Log.log_groundseg(f"{urb.pier_name}: Starting!")
                return 200
            return 400
        
        return 400

    # Toggle Pier Wireguard connection on or off
    def toggle_pier_network(self, urb):


        wg_reg = self.config['wgRegistered']
        wg_is_running = self.wireguard.wg_docker.is_running()
        cfg = urb.config

        network = 'none'
        if cfg['network'] == 'none' and wg_reg and wg_is_running:
            network = 'wireguard'

        x = urb.set_network(network)
        if x == 0:
            Log.log_groundseg(f"{urb.pier_name}: Setting network to {network}")
            return 200

        return 400
            
    # Create minIO admin acccount
    def create_minio_admin_account(self, patp, password):

        self._urbits[patp].config['minio_password'] = password
        self._urbits[patp].save_config()
        self._minios[patp] = MinIODocker(self._urbits[patp].config)
        self._mc.mc_setup(patp, self._minios[patp].config['wg_s3_port'], self._minios[patp].config['minio_password'])
        self.minIO_on = True

        Log.log_groundseg(f"{patp}: Created MinIO admin account")
        return 200
    
    # Get +code from Urbit
    def get_urbit_code(self, patp, urb):
        code = ''
        addr = self.get_urbit_loopback_addr(patp)
        try:
            code = urb.get_code(addr)
        except Exception as e:
            Log.log_groundseg(e)

        return code
 
    # Get looback address of Urbit Pier
    def get_urbit_loopback_addr(self, patp):
        log = self.get_log_lines(patp,0)[::-1]
        substr = 'http: loopback live on'

        for ln in log:
            if substr in ln:
                return str(ln.split(' ')[-1])

    # Boot new pier from key
    def boot_new_urbit(self, patp, key):
        if patp == None:
            return 400

        http_port, ames_port = self.get_open_urbit_ports()
        urb = copy.deepcopy(default_pier_config)

        urb['pier_name'] = patp
        urb['http_port'] = http_port
        urb['ames_port'] = ames_port

        with open(f'/opt/nativeplanet/groundseg/settings/pier/{patp}.json', 'w') as f:
            json.dump(urb, f, indent = 4)
    
        urbit = UrbitDocker(urb)
        urbit.add_key(key)
        x = self.add_urbit(patp, urbit)

        Log.log_groundseg(f"{patp}: Booting new pier")
        return x

    def boot_existing_urbit(self, filename):
        patp = filename.split('.')[0]

        if patp == None:
            return "File is invalid"

        Log.log_groundseg(f"{patp}: Booting existing pier")
        return self.extract_pier(filename)

    def extract_pier(self, filename):
        patp = filename.split('.')[0]
        vol_dir = f'/var/lib/docker/volumes/{patp}'
        compressed_dir = f"{self.config['CFG_DIR']}/uploaded/{patp}/{filename}"

        try:
            Log.log_groundseg(f"{patp}: Removing existing volume")
            os.system(f'rm -rf {vol_dir}')

            Log.log_groundseg(f"{patp}: Creating volume directory")
            os.system(f'mkdir -p {vol_dir}/_data')

            Log.log_groundseg(f"{patp}: Extracting {filename}")
            if filename.endswith("zip"):
                with zipfile.ZipFile(compressed_dir) as zip_ref:
                    zip_ref.extractall(f"{vol_dir}/_data")

            elif filename.endswith("tar.gz") or filename.endswith("tgz") or filename.endswith("tar"):
                tar = tarfile.open(compressed_dir,"r:gz")
                tar.extractall(f"{vol_dir}/_data")
                tar.close()

        except Exception as e:
            Log.log_groundseg(e)
            return "File extraction failed"

        try:
            shutil.rmtree(f"{self.config['CFG_DIR']}/uploaded/{patp}", ignore_errors=True)
            Log.log_groundseg(f"{patp}: Deleted {filename}")

        except Exception as e:
            Log.log_groundseg(e)
            return f"Failed to remove {filename}"

        return self.build_urbit_container_existing(patp)

    def build_urbit_container_existing(self, patp):

        try:
            Log.log_groundseg(f"{patp}: Building docker container")

            http_port, ames_port = self.get_open_urbit_ports()
            data = copy.deepcopy(default_pier_config)

            data['pier_name'] = patp
            data['http_port'] = http_port
            data['ames_port'] = ames_port

            urbit = UrbitDocker(data)

            with open(f'/opt/nativeplanet/groundseg/settings/pier/{patp}.json', 'w') as f:
                json.dump(data, f, indent = 4)

            x = self.add_urbit(patp, urbit)
            if self.config['updateMode'] == 'temp':
                self.config['updateMode'] = 'auto'

            if x == 0:
                return 200

        except Exception as e:
            Log.log_groundseg(f"{patp}: {e}")

        return "Failed to create Urbit pier"

    # Get unused ports for Urbit
    def get_open_urbit_ports(self):
        http_port = 8080
        ames_port = 34343

        for u in self._urbits.values():
            if(u.config['http_port'] >= http_port):
                http_port = u.config['http_port']
            if(u.config['ames_port'] >= ames_port):
                ames_port = u.config['ames_port']

        return http_port+1, ames_port+1

    # Add Urbit to list of Urbit
    def add_urbit(self, patp, urbit):
        self.config['piers'] = [i for i in self.config['piers'] if i != patp]
        self.config['piers'].append(patp)
        urbit.config['boot_status'] = 'boot'
        self._urbits[patp] = urbit

        self.register_urbit(patp)
        
        if self.wireguard.is_running() and len(self.config['piers']) < 2:
            Log.log_groundseg(f"Restarting Anchor")
            x = self.toggle_anchor_off()
            if x == 200:
                self.toggle_anchor_on()

        self.save_config()
        return urbit.start()
 
    # Register Wireguard for Urbit
    def register_urbit(self, patp):
        
        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f'https://{endpoint}/{api_version}'

        if self.config['wgRegistered']:
            x = self.wireguard.get_status(url)
            if x != None:
                self.anchor_config = x 
                self.wireguard.update_wg_config(x['conf'])

            if self.anchor_config != None:
                # Define services
                urbit_web = False
                urbit_ames = False
                minio_svc = False
                minio_console = False
                minio_bucket = False
                # Check if service exists for patp
                for ep in self.anchor_config['subdomains']:
                    ep_patp = ep['url'].split('.')[-3]
                    ep_svc = ep['svc_type']
                    if ep_patp == patp:
                        if ep_svc == 'urbit-web':
                            urbit_web = True
                        if ep_svc == 'urbit-ames':
                            urbit_ames = True
                        if ep_svc == 'minio':
                            minio_svc = True
                        if ep_svc == 'minio-console':
                            minio_console = True
                        if ep_svc == 'minio-bucket':
                            minio_bucket = True
 
                # One or more of the urbit services is not registered
                if not (urbit_web and urbit_ames):
                    Log.log_groundseg(f"{patp}: Registering Urbit anchor services")
                    self.wireguard.register_service(f'{patp}','urbit',url)
 
                # One or more of the minio services is not registered
                if not (minio_svc and minio_console and minio_bucket):
                    Log.log_groundseg(f"{patp}: Registering MinIO anchor services")
                    self.wireguard.register_service(f's3.{patp}','minio',url)

            x = self.wireguard.get_status(url)
            if x != None:
                self.anchor_config = x 
                self.wireguard.update_wg_config(x['conf'])

            url = None
            http_port = None
            ames_port = None
            s3_port = None
            console_port = None

            Log.log_groundseg(self.anchor_config['subdomains'])
            pub_url = '.'.join(self.config['endpointUrl'].split('.')[1:])

            for ep in self.anchor_config['subdomains']:
                if(f'{patp}.{pub_url}' == ep['url']):
                    url = ep['url']
                    http_port = ep['port']
                elif(f'ames.{patp}.{pub_url}' == ep['url']):
                    ames_port = ep['port']
                elif(f'bucket.s3.{patp}.{pub_url}' == ep['url']):
                    s3_port = ep['port']
                elif(f'console.s3.{patp}.{pub_url}' == ep['url']):
                    console_port = ep['port']

            self._urbits[patp].set_wireguard_network(url, http_port, ames_port, s3_port, console_port)
            return self.wireguard.start()

    # Update/Set Urbit S3 Endpoint
    def set_minio_endpoint(self, patp):
        acc = 'urbit_minio'
        secret = ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(40))

        res = self._mc.make_service_account(patp, acc, secret)
        u = self._urbits[patp]

        endpoint = f"s3.{u.config['wg_url']}"
        bucket = 'bucket'
        lens_port = self.get_urbit_loopback_addr(patp)
        
        if res == 200:
            try:
                return u.set_minio_endpoint(endpoint,acc,secret,bucket,lens_port)

            except Exception as e:
                Log.log_groundseg(e)

        return 400

    # Export contents of minIO bucket
    def export_minio_bucket(self, patp):
        m = self._minios[patp]
    
        if(m==None):
            return 400
        
        file_name = f'bucket_{patp}.zip'
        bucket_path=f'/var/lib/docker/volumes/minio_{patp}/_data/bucket'
        shutil.make_archive(f'/app/tmp/bucket_{patp}', 'zip', bucket_path)

        return send_file(f'/app/tmp/{file_name}', download_name=file_name, as_attachment=True)


#
#   Anchor Settings
#
    # Get anchor registration information
    def get_anchor_settings(self):

        lease_end = None
        ongoing = False
        lease = self.anchor_config['lease']

        if self.anchor_config['ongoing'] == 1:
            ongoing = True

        if lease != None:
            x = list(map(int,lease.split('-')))
            lease_end = datetime(x[0], x[1], x[2], 0, 0)

        anchor = dict()
        anchor['wgReg'] = self.config['wgRegistered']
        anchor['wgRunning'] = self.wireguard.is_running()
        anchor['lease'] = lease_end
        anchor['ongoing'] = ongoing

        return {'anchor': anchor}


#
#   System Settings
#
    # Get all system information
    def get_system_settings(self):
        settings = dict()
        settings['vm'] = self._vm
        settings['gsVersion'] = self.gs_version
        settings['updateMode'] = self.config['updateMode']
        settings['minio'] = self.minIO_on
        settings['containers'] = self.get_containers()
        settings['sessions'] = len(self.config['sessions'])

        if not self._vm:
            settings['ram'] = self._ram
            settings['cpu'] = self._cpu
            settings['temp'] = self._core_temp
            settings['disk'] = self._disk
            settings['connected'] = self.get_connection_status()
            settings['ethOnly'] = self.get_ethernet_status()

        return {'system': settings}

    # Modify system settings
    def handle_module_post_request(self, module, data, sessionid):

        # sessions module
        if module == 'session':
            if data['action'] == 'logout':
                self.config['sessions'] = [i for i in self.config['sessions'] if i != sessionid]
                self.save_config()
                return 200

            if data['action'] == 'logout-all':
                self.config['sessions'] = []
                self.save_config()
                return 200

            if data['action'] == 'change-pass':
                encoded_str = (self.config['salt'] + data['old-pass']).encode('utf-8')
                this_hash = hashlib.sha512(encoded_str).hexdigest()

                if this_hash == self.config['pwHash']:
                    self.create_password(data['new-pass'])
                    return 200

        # anchor module
        if module == 'anchor':
            if data['action'] == 'register':
                return self.register_device(data['key']) 

            if data['action'] == 'toggle':
                running = self.wireguard.is_running()
                if running:
                    return self.toggle_anchor_off()
                return self.toggle_anchor_on()

            if data['action'] == 'get-url':
                return self.config['endpointUrl']

            if data['action'] == 'change-url':
                return self.change_wireguard_url(data['url'])

            if data['action'] == 'unsubscribe':
                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f'https://{endpoint}/{api_version}'
                x = self.wireguard.cancel_subscription(data['key'],url)
                if x != 400:
                    self.anchor_config = x
                    return 200


        # power module
        if module == 'power':
            if data['action'] == 'shutdown':
                return self.shutdown()

            if data['action'] == 'restart':
                return self.restart()

        # watchtower module
        if module == 'watchtower':
            if data['action'] == 'toggle':
                return self.set_update_mode()

        # minIO module
        if module == 'minio':
            if data['action'] == 'reload':
                self.toggle_minios_off()
                self.toggle_minios_on()
                time.sleep(1)
                return 200
        
        # network connectivity module
        if module == 'network':
            if data['action'] == 'toggle':
                return self.toggle_ethernet()

            if data['action'] == 'networks':
                if self._vm:
                    return []
                return Network.list_wifi_ssids()

            if data['action'] == 'connect':
                return self.change_wifi_network(data['network'], data['password'])

        # logs module
        if module == 'logs':
            if data['action'] == 'view':
                return self.get_log_lines(data['container'], data['haveLine'])

            if data['action'] == 'export':
                return '\n'.join(self.get_log_lines(data['container'], 0))

        # binary module
        if module == 'binary':
            if data['action'] == 'restart':

                Log.log_groundseg("Restarting groundseg...")

                if sys.platform == "darwin":
                    os.system("launchctl load /Library/LaunchDaemons/io.nativeplanet.groundseg.plist")
                    return 200

                elif sys.platform == "linux":
                    os.system("systemctl restart groundseg")
                    return 200

                else:
                    Log.log_groundseg("Windows not supported yet")

        return module

    # Shutdown
    def shutdown(self):
        if sys.platform == "win32":
            os.system("shutdown /s /t 0")
        else:
            os.system("shutdown -h now")
        return 200

    # Restart
    def restart(self):
        if sys.platform == "win32":
            os.system("shutdown /s /t 0")
        else:
            os.system("reboot")
        return 200

    # Get list of available docker containers
    def get_containers(self):
        minio = list(self._minios.keys())
        containers = list(self._urbits.keys())
        containers.append('wireguard')
        for m in minio:
            containers.append(f"minio_{m}")

        return containers

    # Check if wifi is disabled
    def get_ethernet_status(self):
        wifi_status = subprocess.Popen(['nmcli','radio','wifi'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
        ws, stderr = wifi_status.communicate()

        eth = True
        if ws == b'enabled\n':
            eth = False
        
        self.eth_only = eth
        return eth
    
    # Check if wifi is connected
    def get_connection_status(self):
        check_connected = subprocess.Popen(['nmcli', '-t', 'con', 'show'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
        connections, stderr = check_connected.communicate()
        connections_arr = connections.decode('utf-8').split('\n')
        substr = 'wireless'

        for ln in connections_arr:
            if substr in ln:
                conn = ln.split(':')
                if len(conn[-1]) > 0:
                    return conn[0]
        return ''

    # Enables and disables wifi on the host device
    def toggle_ethernet(self):
        if self.eth_only:
            os.system('nmcli radio wifi on')
        else:
            os.system('nmcli radio wifi off')

        return 200

    def change_wifi_network(self, network, password):
        connect_attempt = subprocess.Popen(['nmcli','dev','wifi','connect',network,'password',password],
                stdout=subprocess.PIPE,stderr=subprocess.STDOUT)

        did_connect, stderr = connect_attempt.communicate()
        did_connect = did_connect.decode("utf-8")[0:5]

        if did_connect == 'Error':
            return 400

        return 200

    # Starts Wireguard and all MinIO containers
    def toggle_anchor_on(self):
        self.wireguard.start()
        self.toggle_minios_on() 
        self.config['wgOn'] = True
        self.save_config()

        Log.log_groundseg("Getting updated wireguard config")

        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f'https://{endpoint}/{api_version}'

        x = self.wireguard.get_status(url)
        if x != None:
            self.anchor_config = x 
            self.wireguard.update_wg_config(x['conf'])

        return 200

    # Stops Wireguard and all MinIO containers
    def toggle_anchor_off(self):
        for p in self._urbits.keys():
            if(self._urbits[p].config['network'] == 'wireguard'):
                 self.toggle_pier_network(self._urbits[p])

        self.toggle_minios_off()
        self.wireguard.stop()
        self.config['wgOn'] = False
        self.save_config()
        return 200

    # Toggle MinIO on
    def toggle_minios_on(self):
      for m in self._minios.values():
         m.start()
      time.sleep(2)
      self.minIO_on = True
      return self.minIO_on

    # Toggl MinIO off
    def toggle_minios_off(self):
      for m in self._minios.values():
         m.stop()
      self.minIO_on = False

    # Register device to an Anchor service using a key
    def register_device(self, reg_key):

        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f'https://{endpoint}/{api_version}'

        x = self.wireguard.register_device(reg_key, url) 

        if x == None:
            return 400

        Log.log_groundseg(f"/register response: {x}")

        if x['error'] != 0:
            return 400

        x = self.wireguard.get_status(url)
        if x != None:
            self.anchor_config = x 
            self.wireguard.update_wg_config(x['conf'])

        if self.anchor_config != None:
            Log.log_groundseg(f"Starting Wireguard")
            self.wireguard.start()
            self.config['wgRegistered'] = True
            self.config['wgOn'] = True
            time.sleep(2)
           
            Log.log_groundseg(f"Registering Urbits")
            for p in self.config['piers']:
                self.register_urbit(p)

            Log.log_groundseg(f"Starting MinIOs")
            self.toggle_minios_on()
            self.save_config()

            time.sleep(2)

            if self.wireguard.is_running() and len(self.config['piers']) > 0:
                Log.log_groundseg(f"Restarting Anchor")
                x = self.toggle_anchor_off()
                if x == 200:
                    self.toggle_anchor_on()

            return 200

        return 400

    # Change Anchor endpoint URL
    def change_wireguard_url(self, url):
        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        old_url = f'https://{endpoint}/{api_version}'
        self.config['endpointUrl'] = url
        self.config['wgRegistered'] = False
        self.config['wgOn'] = False

        for patp in self.config['piers']:
            self.wireguard.delete_service(f'{patp}','urbit',old_url)
            self.wireguard.delete_service(f's3.{patp}','minio',old_url)

        self.toggle_anchor_off()
        self.reset_pubkey()
        self.save_config()
        if self.config['endpointUrl'] == url:
            return 200
        return 400

    # Toggle update mode
    def set_update_mode(self):
        if self.config['updateMode'] == 'auto':
            self.config['updateMode'] = 'off'
        else:
            self.config['updateMode'] = 'auto'

        self.save_config()

        return 200


#
#   General
#

    def save_config(self):
        with open(self.config_file, 'w+') as f:
            json.dump(self.config, f, indent = 4)

    # Reset Public and Private Keys
    def reset_pubkey(self):
        x = WgKey()

        b64pub = x.pubkey + '\n' 
        b64pub = b64pub.encode('utf-8')
        b64pub = base64.b64encode(b64pub).decode('utf-8')

        # Load priv and pub key
        self.config['pubkey'] = b64pub
        self.config['privkey'] = x.privkey

    # Get logs from docker container
    def get_log_lines(self, container, line):

        blob = ''

        if container == 'wireguard':
            blob = self.wireguard.wg_docker.logs()

        if 'minio_' in container:
            blob = self._minios[container[6:]].logs()

        if container in self._urbits.keys():
            blob = self._urbits[container].logs()

        return blob.decode('utf-8').split('\n')[line:]

    # Create new password
    def create_password(self, pwd):
        # create salt
        salt = ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(16))

        # make hash
        encoded_str = (salt + pwd).encode('utf-8')
        hashed = hashlib.sha512(encoded_str).hexdigest()

        # add to config
        self.config['salt'] = salt
        self.config['pwHash'] = hashed
        self.save_config()

    # Custom jsonify
    def custom_jsonify(self, val):
        if type(val) is int:
            return jsonify(val)
        if type(val) is str:
            return jsonify(val)
        return val

