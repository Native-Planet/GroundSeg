# Python
import os
import copy
import time
import json
import socket
import shutil
import string
import secrets
import zipfile
import tarfile

from io import BytesIO
from time import sleep
from pathlib import Path
from threading import Thread
from datetime import datetime

'''
# Flask
from flask import send_file

# GroundSeg Modules
from log import Log
from utils import Utils
'''
from lib.click_wrapper import Click

from groundseg.docker.urbit import UrbitDocker

default_pier_config = {
        "pier_name":"",
        "http_port":8080,
        "ames_port":34343,
        "loom_size":31,
        "urbit_version":"v2.1",
        "minio_version":"latest",
        "urbit_repo": "registry.hub.docker.com/nativeplanet/urbit",
        "minio_repo": "registry.hub.docker.com/minio/minio",
        "urbit_amd64_sha256": "f08cd1717c6191a277f7cee46807eee8d772d8001c61c0610a53455bd56c5a77",
        "urbit_arm64_sha256": "ed13a935b30e9a7666686c153464748ff2b768f1803df39c56ead2cf9e9c29df",
        "minio_amd64_sha256": "f6a3001a765dc59a8e365149ade0ea628494230e984891877ead016eb24ba9a9",
        "minio_arm64_sha256": "567779c9f29aca670f84d066051290faeaae6c3ad3a3b7062de4936aaab2a29d",
        "minio_password": "",
        "network":"none",
        "wg_url": "nan",
        "wg_http_port": None,
        "wg_ames_port": None,
        "wg_s3_port": None,
        "wg_console_port": None,
        "meld_schedule": False,
        "meld_frequency": 7,
        "meld_time": "0000",
        "meld_last": "0",
        "meld_next": "0",
        "boot_status": "boot",
        "custom_urbit_web": '',
        "custom_s3_web": '',
        "show_urbit_web": 'default',
        "dev_mode": False,
        "click": False
        }


class Urbit:

    _volume_directory = '/var/lib/docker/volumes'
    ready = False
    system_info = {}
    vere_version = {}

    def __init__(self, parent, cfg, wg, minio):
        self.app = parent
        self.cfg = cfg
        self.wg = wg
        self.minio = minio

        # Volume directory of urbit ships
        self._volume_directory = f"{self.cfg.system.get('dockerData')}/volumes"

        self._urbits = {}
        self.urb_docker = UrbitDocker(self.cfg)

        # Updater Urbit information
        # Here we update the minio version_info as well as it is tied to the ship
        branch = self.cfg.system.get('updateBranch')
        if self.cfg.version_server_ready and self.cfg.system.get('updateMode') == 'auto':
            self.version_info = self.cfg.version_info['groundseg'][branch]['vere']
            self.minio_version_info = self.cfg.version_info['groundseg'][branch]['minio']

        # Start everything
        self.start_all(self.cfg.system.get('piers'))

        print("groundseg:urbit:init Initialization Completed")
        self.ready = True

    # Start container
    def start(self, patp, key='', skip=False):
        # We are able to skip the load_config
        # See docker_updater
        if not skip:
            skip = self.load_config(patp)
        if skip:
            if self.minio.start_minio(f"minio_{patp}", self._urbits[patp]):
                return self.urb_docker.start(self._urbits[patp],
                                             self.cfg.arch,
                                             self._volume_directory,
                                             key
                                             )
        return "failed"

    def ram(self):
        for p in self.cfg.system.get('piers').copy():
            self.system_info[p] = self.urb_docker.get_memory_usage(p)


    def set_vere_version(self,patp,version):
        self.vere_version[patp] = version

    def stop(self, patp):
        self.urb_docker.exec(patp, f"cat {patp}/.vere.lock")
        if self.urb_docker.exec(patp, f"kill $(cat {patp}/.vere.lock"):
            self.urb_docker.exec(patp, f"cat {patp}/.vere.lock")
        if self.graceful_exit(patp):
            return self.urb_docker.stop(patp)

    # |exit 
    def graceful_exit(self, patp):
        try:
            print(f"{patp}: Attempting to send |exit")
            # Naming the hoon file
            name = "bar_exit"
            ";<  our=@p  bind:m  get-our"
            hoon = f"=/  m  (strand ,vase)  ;<  ~  bind:m  (poke [~{patp} %hood] %drum-exit !>(~))  (pure:m !>('success'))"
            hoon_file = f"{name}.hoon"
            self.create_hoon(patp, name, hoon)
            # Executing the hoon file
            raw = Click().click_exec(patp, self.urb_docker.exec, hoon_file)
            res = Click().filter_success(raw)
            self.delete_hoon(patp, name)
            print(f"{patp}: |exit sent successfully")
        except Exception as e:
            print(f"urbit:graceful_exit:{patp} Error: {e}")
            return False
        return True

    '''
    # Delete Urbit Pier and MiniO
    def delete(self, patp):
        print(f"{patp}: Attempting to delete all data")
        try:
            if self.urb_docker.delete(patp):

                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f'https://{endpoint}/{api_version}'

                if self.config['wgRegistered']:
                    self.wg.delete_service(f'{patp}','urbit',url)
                    self.wg.delete_service(f's3.{patp}','minio',url)

                self.minio.delete(f"minio_{patp}")

                print(f"{patp}: Deleting from system.json")
                self.config['piers'] = [i for i in self.config['piers'] if i != patp]
                self.config_object.save_config()

                print(f"{patp}: Removing {patp}.json")
                os.remove(f"/opt/nativeplanet/groundseg/settings/pier/{patp}.json")

                self._urbits.pop(patp)
                print(f"{patp}: Data removed from GroundSeg")

                return 200

        except Exception as e:
            print(f"{patp}: Failed to delete data: {e}")

        return 400
    '''
    '''

    def export(self, patp):
        print(f"{patp}: Attempting to export pier")
        c = self.urb_docker.get_container(patp)
        if c:
            if c.status == "running":
                self.stop(patp)

            file_name = f"{patp}.zip"
            memory_file = BytesIO()
            file_path=f"{self._volume_directory}/{patp}/_data/"

            print(f"{patp}: Compressing pier")

            with zipfile.ZipFile(memory_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
                for root, dirs, files in os.walk(file_path):
                    arc_dir = root[root.find("_data/")+6:]
                    for file in files:
                        if file != 'conn.sock':
                            zipf.write(os.path.join(root, file), arcname=os.path.join(arc_dir,file))
                        else:
                            print(f"{patp}: Skipping {file} while compressing")

            memory_file.seek(0)

            print(f"{patp}: Pier successfully exported")
            return send_file(memory_file, download_name=file_name, as_attachment=True)

    '''

    # Start all valid containers
    def start_all(self, patps):
        print("groundseg:urbit:start_all Starting all ships")
        # Ships will fall into one of these categories:
        res = {
                "failed":[],    # Container is not running.
                "succeeded":[], # Docker confirmed that container is running.
                "ignored":[],   # boot_status is set to ingore.
                "invalid":[]    # not a valid ship.
                }

        # No ships
        if len(patps) < 1:
            print("Urbit: No ships detected in system.json! Skipping..")
            return True

        # We loop through the list of @p and start them individually 
        for p in patps:
            status = self.start(p)
            try:
                res[status].append(p)
            except Exception as e:
                print(f"{p}: {e}")

        print(f"Urbit: Start succeeded {res['succeeded']}")
        print(f"Urbit: Start ignored {res['ignored']}")
        print(f"Urbit: Start failed {res['failed']}")
        print(f"Urbit: Patp invalid {res['invalid']}")

        return True

    # Boot new pier from key
    async def create(self, patp, key, remote):
        print(f"groundseg:urbit:{patp}:create Attempting to boot new urbit ship")
        try:
            if not self.cfg.check_patp(patp):
                raise Exception("Invalid @p")

            # TODO: Add check if exists, return prompt to user for further action

            # Get open ports
            http_port, ames_port = self.get_open_urbit_ports()

            # Generate config file for pier
            cfg = self.build_config(patp, http_port, ames_port)
            self._urbits[patp] = cfg

            self.save_config(patp)
            # Delete existing ship if exists
            if self.urb_docker.delete(patp):
                # Add to system.json
                if self.cfg.add_system_patp(patp):
                    # Boot ship
                    if self.start(patp, key) == "succeeded":
                        # toggle to remote if required
                        if remote:
                            Thread(target=self.new_pier_remote_toggle,args=(patp,)).start()
            return True
        except Exception as e:
            print(f"groundseg:urbit:{patp}:create: Failed to boot new urbit ship: {e}")
        return False

    def new_pier_remote_toggle(self, patp):
        print(f"groundseg:urbit:{patp}:new_pier_remote_toggle: New pier remote toggle thread started")
        try:
            running = self.urb_docker.is_running(patp)
            booted = len(self.get_code(patp)) == 27
            count = 0
            registered = self.check_services_registered(patp)

            while not (running and booted and registered):
                print(f"groundseg:urbit:{patp}:new_pier_remote_toggle: Ship not ready for remote toggle yet")
                time.sleep(count * 2)
                if count < 5:
                    count += 1
                running = self.urb_docker.is_running(patp)
                booted = len(self.get_code(patp)) == 27
                registered = self.check_services_registered(patp)
            self.toggle_network(patp)
        except Exception as e:
            print(f"groundseg:urbit:{patp}:new_pier_remote_toggle: Failed to start new pier remote toggle thread: {e}")

    def check_services_registered(self, patp):
        registered = False
        try:
            if self.app.startram.retrieve_status(1):
                registered = True
                service_status = self.wg.anchor_services.get(patp)
                for svc in service_status.keys():
                    try:
                        if service_status[svc]['status'] != "ok":
                            raise Exception(f"{svc} not ready")
                    except Exception as e:
                        raise Exception(e)
        except Exception as e:
            registered = False
            print("groundseg:urbit:{patp}:check_services_registered: Error {e}")
        return registered

    def fix_pokes(self, patp):
        print(f"{patp}: Pier upload fix pokes thread started")
        try:
            running = self.urb_docker.is_running(patp)
            booted = len(self.get_code(patp)) == 27
            count = 0
            while not (running and booted):
                print(f"{patp}: Ship not ready for pokes yet")
                time.sleep(count * 2)
                if count < 5:
                    count += 1
                running = self.urb_docker.is_running(patp)
                booted = len(self.get_code(patp)) == 27
            self.fix_acme(patp)
        except Exception as e:
            print(f"{patp}: Failed to start fix pokes thread: {e}")

    def boot_existing(self, filename, remote, fix, create_service):
        print(f"groundseg:urbit:boot_existing Configuration - remote: {remote} - fix: {fix}")
        patp = filename.split('.')[0]
 
        # Make sure patp is valid
        if not self.cfg.check_patp(patp):
            return "Invalid @p"

        # Extract the pier
        extracted = self.extract_pier(filename)
        if extracted != "to-create":
            return extracted

        # Create the groundseg ship
        created = self.create_existing(patp)
        if created != "succeeded":
            return created

        # register services
        if self.cfg.system.get('wgRegistered'):
            create_service(patp, 'urbit')
            create_service(f"s3.{patp}", 'minio')

        # Au70 70ggL3z
        if remote:
            Thread(target=self.new_pier_remote_toggle, args=(patp,), daemon=True).start()

        # f!x0rZ
        if fix:
            Thread(target=self.fix_pokes, args=(patp,), daemon=True).start()

        return 200

    def extract_pier(self, filename):
        patp = filename.split('.')[0]
        vol_dir = f'{self._volume_directory}/{patp}'
        compressed_dir = f"{self.cfg.base}/uploaded/{patp}/{filename}"

        try:
            # Remove directory and make new empty one
            print(f"{patp}: Removing existing volume")
            shutil.rmtree(f"{vol_dir}", ignore_errors=True)
            print(f"{patp}: Creating volume directory")
            os.system(f'mkdir -p {vol_dir}/_data')

            # Begin extraction
            print(f"{patp}: Extracting {filename}")

            # Zipfile
            if filename.endswith("zip"):
                with zipfile.ZipFile(compressed_dir) as zip_ref:
                    total_size = sum((file.file_size for file in zip_ref.infolist()))
                    zip_ref.extractall(f"{vol_dir}/_data")

            # Tarball
            elif filename.endswith("tar.gz") or filename.endswith("tgz") or filename.endswith("tar"):
                with tarfile.open(compressed_dir, "r") as tar_ref:
                    total_size = sum((member.size for member in tar_ref.getmembers()))
                    tar_ref.extractall(f"{vol_dir}/_data")

        except Exception as e:
            print(f"{patp}: Failed to extract {filename}: {e}")
            return "File extraction failed"

        # Restructure directory
        try:
            # Get all .urb locations in directory
            data_dir = os.path.join(vol_dir, '_data')
            urb_loc = []
            for root, dirs, files in os.walk(data_dir):
                if ('.urb' in dirs) and ('__MACOSX' not in root):
                    urb_loc.append(root)

            # Fail if more than one .urb exists
            if len(urb_loc) > 1:
                text = f"Multiple ships ({len(urb_loc)}) detected in pier directory"
                print(f"{patp}: {text}")
                return text
            if len(urb_loc) < 1:
                print(f"{patp}: No ships detected in pier directory")
                return "No Urbit ship found in pier directory"

            print(f"{patp}: .urb subdirectory in {urb_loc[0]}")

            pier_dir = os.path.join(data_dir, patp)
            temp_dir = os.path.join(data_dir, 'temp_dir')
            unused_dir = os.path.join(data_dir, 'unused')

            # check if .urb is in the correct location 
            if os.path.join(pier_dir, '.urb') != os.path.join(urb_loc[0], '.urb'):
                print(f"{patp}: .urb location incorrect!")
                print(f"{patp}: Restructuring directory structure")

                # move to temp dir
                print(f"{patp}: .urb found in {urb_loc[0]}")
                print(f"{patp}: Moving to {temp_dir}")
                if data_dir == urb_loc[0]: # .urb in root
                    # Create directory
                    os.makedirs(temp_dir, exist_ok=True)
                    # select everything in root except for pier_dir
                    items = [x for x in list(Path(urb_loc[0]).iterdir()) if str(x) != pier_dir]
                    print(f"{patp}: Items to move: {items}")
                    for item in items:
                        shutil.move(str(item), temp_dir)
                else:
                    shutil.move(urb_loc[0], temp_dir)

                # rename directories
                unused = [str(x) for x in list(Path(data_dir).iterdir()) if (str(x) != temp_dir) and (str(x) != unused_dir)]
                if len(unused) > 0:
                    # Create directory
                    os.makedirs(unused_dir, exist_ok=True)
                    print(f"{patp}: Moving unused items to {unused_dir}")
                    for u in unused:
                        print(f"{patp}: Unused items to move: {unused}")
                        shutil.move(u, unused_dir)

                shutil.move(temp_dir, pier_dir)

                print(f"{patp}: Restructuring done!")
            else:
                print(f"{patp}: No restructuring needed!")

        except Exception as e:
            print(f"{patp}: Failed to restructure directory: {e}")
            return f"Failed to restructure {patp}"

        try:
            shutil.rmtree(f"{self.cfg.base}/uploaded/{patp}", ignore_errors=True)
            print(f"{patp}: Deleted {filename}")

        except Exception as e:
            print(f"{patp}: Failed to remove {filename}: {e}")
            return f"Failed to remove {filename}"

        return "to-create"

    # Boot the newly uploaded pier
    def create_existing(self, patp):
        print(f"groundseg:urbit:{patp}:create_existing Attempting to boot uploaded urbit ship")
        try:
            # Get open ports
            http_port, ames_port = self.get_open_urbit_ports()

            # Generate config file for pier
            cfg = self.build_config(patp, http_port, ames_port)
            self._urbits[patp] = cfg
            self.save_config(patp)

            # Add to system.json
            if self.cfg.add_system_patp(patp):
                # Boot ship
                return self.start(patp)

        except Exception as e:
            print(f"{patp}: Failed to boot uploaded urbit ship: {e}")

        return f"Failed to boot {patp}"

    '''

   # Return all details of Urbit ID
    def get_info(self, patp):
        # Check if Urbit Pier exists
        c = self.urb_docker.get_container(patp)
        if c:
            # If MinIO container exists
            containers = [patp]
            has_bucket = False
            if self.minio.minio_docker.get_container(f"minio_{patp}", False):
                containers.append(f"minio_{patp}")
                has_bucket = True

            cfg = self._urbits[patp]
            urbit = {
                "name": patp,
                "running": c.status == "running",
                "wgReg": self.config['wgRegistered'],
                "wgRunning": self.wg.is_running(),
                "autoboot": cfg['boot_status'] != 'off',
                "meldOn": cfg['meld_schedule'],
                "timeNow": datetime.utcnow(),
                "frequency": cfg['meld_frequency'],
                "meldLast": datetime.fromtimestamp(int(cfg['meld_last'])),
                "meldNext": datetime.fromtimestamp(int(cfg['meld_next'])),
                "containers": containers,
                "meldHour": int(cfg['meld_time'][0:2]),
                "meldMinute": int(cfg['meld_time'][2:]),
                "remote": False,
                "urbitUrl": f"http://{socket.gethostname()}.local:{cfg['http_port']}",
                "minIOUrl": "",
                "minIOReg": True,
                "hasBucket": has_bucket,
                "loomSize": cfg['loom_size'],
                "showUrbWeb": 'default',
                "urbWebAlias": cfg['custom_urbit_web'],
                "s3WebAlias": cfg['custom_s3_web'],
                "devMode": cfg['dev_mode'],
                "click": cfg['click']
                }

            if cfg['network'] == 'wireguard':
                urbit['remote'] = True
                urbit['urbitUrl'] = f"https://{cfg['wg_url']}"

                if cfg['show_urbit_web'] == 'alias':
                    if cfg['custom_urbit_web']:
                        urbit['urbitUrl'] = f"https://{cfg['custom_urbit_web']}"
                        urbit['showUrbWeb'] = 'alias'

            if self.config['wgRegistered']:
                urbit['minIOUrl'] = f"https://console.s3.{cfg['wg_url']}"

            if cfg['minio_password'] == '':
                 urbit['minIOReg'] = False

            return urbit
        return 400
        '''


    # Get unused ports for Urbit
    def get_open_urbit_ports(self):
        http_port = 8080
        ames_port = 34343

        for u in self._urbits.values():
            if(u['http_port'] >= http_port):
                http_port = u['http_port']
            if(u['ames_port'] >= ames_port):
                ames_port = u['ames_port']

        return http_port+1, ames_port+1

    # Build new ship config
    def build_config(self, patp, http_port, ames_port):
        urb = copy.deepcopy(default_pier_config)

        urb['pier_name'] = patp
        urb['http_port'] = http_port
        urb['ames_port'] = ames_port

        return urb

    # Toggle Pier on or off
    def toggle_power(self, patp, broadcaster):
        print(f"{patp}: Attempting to toggle container")
        c = self.urb_docker.get_container(patp)
        if c:
            cfg = self._urbits[patp]
            old_status = cfg['boot_status']
            if c.status == "running":
                broadcaster.urbits.set_transition(patp,"togglePower","stopping")
                if self.stop(patp):
                    if cfg['boot_status'] != 'off':
                        self._urbits[patp]['boot_status'] = 'noboot'
                        print(f"{patp}: Boot status changed: {old_status} -> {self._urbits[patp]['boot_status']}")
                        self.save_config(patp)
                        broadcaster.urbits.set_transition(patp,"togglePower","success")
                        sleep(3)
            else:
                broadcaster.urbits.set_transition(patp,"togglePower","booting")
                if cfg['boot_status'] != 'off':
                    self._urbits[patp]['boot_status'] = 'boot'
                    print(f"{patp}: Boot status changed: {old_status} -> {self._urbits[patp]['boot_status']}")
                    self.save_config(patp)
                    if self.start(patp) == "succeeded":
                        broadcaster.urbits.set_transition(patp,"togglePower","success")
                        sleep(3)
        broadcaster.urbits.clear_transition(patp,"togglePower")

    # Create .hoon for pokes
    def create_hoon(self, patp, name, hoon):
        try:
            with open(f'{self._volume_directory}/{patp}/_data/{name}.hoon','w') as f :
                f.write(hoon)
                f.close()
        except Exception:
            print(f"{patp}: Creating {name}.hoon failed")
            return False
        return True

    # Create .hoon for pokes
    def delete_hoon(self, patp, name):
        try:
            hoon_file = f'{self._volume_directory}/{patp}/_data/{name}.hoon'
            if os.path.exists(hoon_file):
                os.remove(hoon_file)
        except Exception:
            print(f"{patp}: Deleting {name}.hoon failed")
            return False
        return True

    # Get +code from Urbit
    def get_code(self, patp):
        name = "code"
        hoon = "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  code=@p  bind:m  (scry @p /j/code/(scot %p our))  (pure:m !>((crip (slag 1 (scow %p code)))))"
        hoon_file = f"{name}.hoon"
        self.create_hoon(patp, name, hoon)
        raw = Click().click_exec(patp, self.urb_docker.exec, hoon_file)
        code = Click().filter_code(raw)
        self._urbits[patp]['click'] = True
        self.delete_hoon(patp, name)

        '''
        if not code:
            self._urbits[patp]['click'] = False
            code = ''
            lens_addr = self.get_loopback_addr(patp)

            try:
                f_data = {"source": {"dojo": "+code"}, "sink": {"stdout": None}}
                with open(f'{self._volume_directory}/{patp}/_data/code.json','w') as f :
                    json.dump(f_data, f)

                command = f'curl -s -X POST -H "Content-Type: application/json" -d @code.json {lens_addr}'
                res = self.urb_docker.exec(patp, command)
                if res:
                    code = res.output.decode('utf-8').strip().split('\\')[0][1:]
                    os.remove(f'{self._volume_directory}/{patp}/_data/code.json')
            except Exception as e:
                print(f"groundseg:urbit:{patp}:get_code Failed to get +code {e}")

        elif code == 'not-yet':
            code = ''
        self.save_config(patp)
        '''
        if not code:
            code = ""
        return code

    # Toggle Autoboot
    def toggle_autoboot(self, patp):
        print(f"{patp}: Attempting to toggle autoboot")
        c = self.urb_docker.get_container(patp)
        if c:
            try:
                cfg = self._urbits[patp]
                old_status = cfg['boot_status']
                if old_status == 'off':
                    if c.status == "running":
                        self._urbits[patp]['boot_status'] = 'boot'
                    else:
                        self._urbits[patp]['boot_status'] = 'noboot'
                else:
                    self._urbits[patp]['boot_status'] = 'off'

                self.save_config(patp)
                print(f"{patp}: Boot status changed: {old_status} -> {self._urbits[patp]['boot_status']}")
                self.save_config(patp)

            except Exception as e:
                print(f"{patp}: Unable to toggle autoboot: {e}")

    def toggle_devmode(self, patp):
        print(f"{patp}: Attempting to toggle developer mode")
        old = self._urbits.get(patp,{}).get('dev_mode')
        new = not old
        print(f"{patp}: Dev mode: {old} -> {new}")
        try:
            self._urbits[patp]['dev_mode'] = new
            if self.urb_docker.remove_container(patp):
                created = self.urb_docker.start(self._urbits[patp],
                                                self.cfg.arch,
                                                self._volume_directory
                                                )
                if created == "succeeded":
                    self.save_config(patp)
                    x = self.start(patp)
                    if not x:
                        raise Exception("start returned {x}")
                else:
                    raise Exception(f"created: {created}")
        except Exception as e:
            print(f"{patp}: Failed to toggle dev mode: {e}")

    def toggle_network(self, patp):
        print(f"{patp}: Attempting to toggle network")

        wg_reg = self.cfg.system.get('wgRegistered')
        wg_is_running = self.wg.is_running()
        c = self.urb_docker.get_container(patp)
        if c:
            try:
                running = False
                if c.status == "running":
                    running = True
                
                old_network = self._urbits[patp]['network']

                self.urb_docker.remove_container(patp)

                if old_network == "none" and wg_reg and wg_is_running:
                    self._urbits[patp]['network'] = "wireguard"
                else:
                    self._urbits[patp]['network'] = "none"

                print(f"groundseg:urbit:{patp}:toggle_network: {old_network} -> {self._urbits[patp]['network']}")
                self.save_config(patp)

                created = self.urb_docker.start(self._urbits[patp],
                                                self.cfg.arch,
                                                self._volume_directory
                                                )
                if (created == "succeeded") and running:
                    self.start(patp)

                return True

            except Exception as e:
                print(f"groundseg:urbit:{patp}:toggle_network: Unable to change network: {e}")

        return False
    '''

    def set_loom(self, patp, size):
        print(f"{patp}: Attempting to set loom size")
        c = self.urb_docker.get_container(patp)
        if c:
            try:
                running = False
                if c.status == "running":
                    running = True
                
                old_loom = self._urbits[patp]['loom_size']
                self.urb_docker.remove_container(patp)
                self._urbits[patp]['loom_size'] = size
                self.save_config(patp)
                print(f"{patp}: Loom size changed: {old_loom} -> {self._urbits[patp]['loom_size']}")

                created = self.urb_docker.start(self._urbits[patp],
                                                self.config_object._arch,
                                                self._volume_directory
                                                )
                if (created == "succeeded") and running:
                    self.start(patp)

                return 200

            except Exception as e:
                print(f"{patp}: Unable to set loom size: {e}")

        return 400

    def schedule_meld(self, patp, freq, hour, minute):
        print(f"{patp}: Attempting to schedule meld frequency")
        try:
            old_sched = self._urbits[patp]['meld_frequency']
            current_meld_next = datetime.fromtimestamp(int(self._urbits[patp]['meld_next']))
            time_replaced_meld_next = int(current_meld_next.replace(hour=hour, minute=minute).timestamp())

            day_difference = freq - self._urbits[patp]['meld_frequency']
            day = 60 * 60 * 24 * day_difference

            self._urbits[patp]['meld_next'] = str(day + time_replaced_meld_next)

            if hour < 10:
                hour = '0' + str(hour)
            else:
                hour = str(hour)

            if minute < 10:
                minute = '0' + str(minute)
            else:
                minute = str(minute)

            self._urbits[patp]['meld_time'] = hour + minute
            self._urbits[patp]['meld_frequency'] = int(freq)

            if self._urbits[patp]['meld_frequency'] > 1:
                days = "days"
            else:
                days = "day"

            print(f"{patp}: Meld frequency changed: {old_sched} Days -> {self._urbits[patp]['meld_frequency']} {days}")
            self.save_config(patp)

            return 200

        except Exception as e:
            print(f"{patp}: Unable to schedule meld: {e}")

        return 400

    def toggle_meld(self, patp):
        print(f"{patp}: Attempting to toggle automatic meld")
        try:
            self._urbits[patp]['meld_schedule'] = not self._urbits[patp]['meld_schedule']
            print(f"{patp}: Automatic meld changed: {not self._urbits[patp]['meld_schedule']} -> {self._urbits[patp]['meld_schedule']}")
            self.save_config(patp)

            try:
                now = int(datetime.utcnow().timestamp())
                if self._urbits[patp]['meld_schedule']:
                    if int(self._urbits[patp]['meld_next']) <= now:
                        self.send_pack_meld(patp)
            except:
                pass

        except Exception as e:
            print(f"{patp}: Unable to toggle automatic meld: {e}")

        return 200

    def send_pack_meld(self, patp):
        lens_addr = self.get_loopback_addr(patp)
        pack = "=/  m  (strand ,vase)  ;<  ~  bind:m  (flog [%pack ~])  (pure:m !>('success'))"
        meld = "=/  m  (strand ,vase)  ;<  ~  bind:m  (flog [%meld ~])  (pure:m !>('success'))"
        if self.send_pack(patp, pack, lens_addr):
            if self.send_meld(patp, meld, lens_addr):
                return 200
        return 400

    def send_pack(self, patp, hoon, lens_addr):
        print(f"{patp}: Attempting to send |pack")
        # Naming the hoon file
        name = "pack"
        hoon_file = f"{name}.hoon"
        self.create_hoon(patp, name, hoon)
        # Executing the hoon file
        raw = Click().click_exec(patp, self.urb_docker.exec, hoon_file)
        pack = Click().filter_success(raw)
        # Set click support to True
        self._urbits[patp]['click'] = True
        # If pack failed
        if not pack:
            try:
                # Set click support to False
                self._urbits[patp]['click'] = False
                data = {"source": {"dojo": "+hood/pack"}, "sink": {"app": "hood"}}
                with open(f'{self._volume_directory}/{patp}/_data/pack.json','w') as f :
                    json.dump(data, f)

                command = f'curl -s -X POST -H "Content-Type: application/json" -d @pack.json {lens_addr}'
                pack = self.urb_docker.exec(patp, command)
            except Exception as e:
                print(f"{patp}: Failed to send |pack: {e}")
                # Set pack to false when error
                pack = False

        # If pack succeeded
        if pack:
            self.delete_hoon(patp, name)
            try:
                os.remove(f'{self._volume_directory}/{patp}/_data/pack.json')
            except:
                pass

            print(f"{patp}: |pack sent successfully")
            self.save_config(patp)
            return True

        return False


    def send_meld(self, patp, hoon, lens_addr):
        print(f"{patp}: Attempting to send |meld")
        # Naming the hoon file
        name = "meld"
        hoon_file = f"{name}.hoon"
        self.create_hoon(patp, name, hoon)
        # Executing the hoon file
        raw = Click().click_exec(patp, self.urb_docker.exec, hoon_file)
        meld = Click().filter_success(raw)
        # Set click support to True
        self._urbits[patp]['click'] = True
        # If meld failed
        if not meld:
            try:
                # Set click support to False
                self._urbits[patp]['click'] = False
                data = {"source": {"dojo": "+hood/meld"}, "sink": {"app": "hood"}}
                with open(f'{self._volume_directory}/{patp}/_data/meld.json','w') as f :
                    json.dump(data, f)

                command = f'curl -s -X POST -H "Content-Type: application/json" -d @pack.json {lens_addr}'
                meld = self.urb_docker.exec(patp, command)
            except Exception:
                print(f"{patp}: Failed to send |meld")
                # Set meld to false when error
                meld = False

        # If meld succeeded
        if meld:
            self.delete_hoon(patp, name)
            try:
                os.remove(f'{self._volume_directory}/{patp}/_data/meld.json')
            except:
                pass

            now = datetime.utcnow()
            self._urbits[patp]['meld_last'] = str(int(now.timestamp()))

            hour, minute = self._urbits[patp]['meld_time'][0:2], self._urbits[patp]['meld_time'][2:]
            meld_next = int(now.replace(hour=int(hour), minute=int(minute), second=0).timestamp())
            day = 60 * 60 * 24 * self._urbits[patp]['meld_frequency']

            self._urbits[patp]['meld_next'] = str(meld_next + day)

            if self._urbits[patp]['meld_frequency'] > 1:
                days = "days"
            else:
                days = "day"

            print(f"{patp}: |meld sent successfully")
            print(f"{patp}: Next meld in {self._urbits[patp]['meld_frequency']} {days}")
            self.save_config(patp)
            return True
        return False
    '''

    # Get looback address of Urbit Pier
    def get_loopback_addr(self, patp):
        log = self.urb_docker.full_logs(patp)
        if log:
            log_arr = log.decode("utf-8").split('\n')[::-1]
            substr = 'http: loopback live on'
            for ln in log_arr:
                if substr in ln:
                    return str(ln.split(' ')[-1])
    '''

    # Register Wireguard for Urbit
    def register_urbit(self, patp, url):
        if self.config['wgRegistered']:
            print(f"{patp}: Attempting to register anchor services")
            if self.wg.get_status(url):
                self.wg.update_wg_config(self.wg.anchor_data['conf'])

                # Define services
                urbit_web = False
                urbit_ames = False
                minio_svc = False
                minio_console = False
                minio_bucket = False

                # Check if service exists for patp
                for ep in self.wg.anchor_data['subdomains']:
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
                    print(f"{patp}: Registering ship")
                    self.wg.register_service(f'{patp}', 'urbit', url)
 
                # One or more of the minio services is not registered
                if not (minio_svc and minio_console and minio_bucket):
                    print(f"{patp}: Registering MinIO")
                    self.wg.register_service(f's3.{patp}', 'minio', url)

            svc_url = None
            http_port = None
            ames_port = None
            s3_port = None
            console_port = None
            tries = 1

            while None in [svc_url,http_port,ames_port,s3_port,console_port]:
                print(f"{patp}: Checking anchor config if services are ready")
                if self.wg.get_status(url):
                    self.wg.update_wg_config(self.wg.anchor_data['conf'])

                print(f"Anchor: {self.wg.anchor_data['subdomains']}")
                pub_url = '.'.join(self.config['endpointUrl'].split('.')[1:])

                for ep in self.wg.anchor_data['subdomains']:
                    if ep['status'] == 'ok':
                        if(f'{patp}.{pub_url}' == ep['url']):
                            svc_url = ep['url']
                            http_port = ep['port']
                        elif(f'ames.{patp}.{pub_url}' == ep['url']):
                            ames_port = ep['port']
                        elif(f'bucket.s3.{patp}.{pub_url}' == ep['url']):
                            s3_port = ep['port']
                        elif(f'console.s3.{patp}.{pub_url}' == ep['url']):
                            console_port = ep['port']
                    else:
                        t = tries * 2
                        print(f"Anchor: {ep['svc_type']} not ready. Trying again in {t} seconds.")
                        time.sleep(t)
                        if tries <= 15:
                            tries = tries + 1
                        break

            return self.set_wireguard_network(patp, svc_url, http_port, ames_port, s3_port, console_port)
        return True

    def set_wireguard_network(self, patp, url, http_port, ames_port, s3_port, console_port):
        print(f"{patp}: Setting wireguard information")
        try:
            self._urbits[patp]['wg_url'] = url
            self._urbits[patp]['wg_http_port'] = http_port
            self._urbits[patp]['wg_ames_port'] = ames_port
            self._urbits[patp]['wg_s3_port'] = s3_port
            self._urbits[patp]['wg_console_port'] = console_port
            return self.save_config(patp)
        except Exception:
            print(f"{patp}: Failed to set wireguard information")
            return False

    # Update/Set Urbit S3 Endpoint

            lens_port = self.get_loopback_addr(patp)
            try:
                return self.set_minio_endpoint(patp, endpoint, acc, secret, bucket, lens_port)

            except Exception as e:
                print(f"{patp}: Failed to set MinIO endpoint: {e}")

        return 400
    '''

    def fix_acme(self, patp):
        lens_addr = self.get_loopback_addr(patp)
        try:
            p_data = {"source": {"dojo": "+hood/pass [%e %rule %cert ~]"}, "sink": {"app": "hood"}}
            with open(f'{self._volume_directory}/{patp}/_data/acmepass.json','w') as f :
                json.dump(p_data, f)

            pass_command = f'curl -s -X POST -H "Content-Type: application/json" -d @acmepass.json {lens_addr}'
            res = self.urb_docker.exec(patp, pass_command)

            os.remove(f'{self._volume_directory}/{patp}/_data/acmepass.json')

            if res.output.decode('utf-8').strip() == '">="':
                print(f"{patp}: acme pass command sent successfully")
                i_data = {"source": {"dojo": "%init"}, "sink": {"app": "acme"}}
                with open(f'{self._volume_directory}/{patp}/_data/acmeinit.json','w') as f :
                    json.dump(i_data, f)

                init_command = f'curl -s -X POST -H "Content-Type: application/json" -d @acmeinit.json {lens_addr}'
                res = self.urb_docker.exec(patp, init_command)

                os.remove(f'{self._volume_directory}/{patp}/_data/acmeinit.json')

                if res.output.decode('utf-8').strip() == '">="':
                    print(f"{patp}: acme init command sent successfully")
                else:
                    print(f"{patp}: Failed to send acme init command")
            else:
                print(f"{patp}: Failed to send acme pass command")

        except Exception as e:
            print(f"{patp}: Failed to clear acme: {e}")

    def update_wireguard_network(self):
        # get new
        new = self.wg.anchor_services.copy()
        # get pier configs
        piers = self._urbits.copy()
        for patp in piers:
            print(f"{patp}: Attempting to update wireguard network")
            changed = False
            cfg = piers[patp]
            services = new.get(patp)
            if services:
                for svc_type in services.keys():
                    service = services.get(svc_type)
                    url = service.get('url')
                    port = service.get('port')
                    alias = service.get('alias')

                    if svc_type == 'urbit-web':
                        if alias == "null":
                            alias = ""
                        if cfg['wg_url'] != url:
                            print(f"{patp}: Wireguard URL changed from {cfg['wg_url']} to {url}")
                            cfg['wg_url'] = url
                            changed = True
                        if cfg['wg_http_port'] != port:
                            print(f"{patp}: Wireguard HTTP Port changed from {cfg['wg_http_port']} to {port}")
                            cfg['wg_http_port'] = port
                            changed = True
                        if cfg['custom_urbit_web'] != alias:
                            print(f"{patp}: Urbit Web Custom URL changed from {cfg['custom_urbit_web']} to {alias}")
                            cfg['custom_urbit_web'] = alias
                            changed = True

                    if svc_type == 'urbit-ames':
                        if cfg['wg_ames_port'] != port:
                            print(f"{patp}: Wireguard Ames Port changed from {cfg['wg_ames_port']} to {port}")
                            cfg['wg_ames_port'] = port
                            changed = True

                    if svc_type == 'minio-bucket':
                        if cfg['wg_s3_port'] != port:
                            print(f"{patp}: Wireguard S3 Port changed from {cfg['wg_s3_port']} to {port}")
                            cfg['wg_s3_port'] = port
                            changed = True

                    if svc_type == 'minio-console':
                        if cfg['wg_console_port'] != port:
                            print(f"{patp}: Wireguard Console Port changed from {cfg['wg_console_port']} to {port}")
                            cfg['wg_console_port'] = port
                            changed = True

                try:
                    if changed:
                        self._urbits[patp] = cfg
                        self.save_config(patp)

                        if cfg['network'] == "wireguard" and self.urb_docker.is_running(patp):
                                # remove minio container
                                self.minio.minio_docker.remove_container(f"minio_{patp}")
                                # remove urbit container
                                if self.urb_docker.remove_container(patp):
                                    # start minio
                                    # start urbit
                                    created = self.urb_docker.start(self._urbits[patp],
                                                                    self.cfg.arch,
                                                                    self._volume_directory
                                                                    )
                                    if created == "succeeded":
                                        self.start(patp)
                                    print(f"{patp}: Wireguard network settings updated!")
                    else:
                        print(f"{patp}: Nothing to change!")
                except Exception as e:
                    print(f"{patp}: Unable to update Wireguard network: {e}")
            else:
                print(f"{patp}: No services found")

    '''
    # Custom Domain
    def custom_domain(self, patp, data):
        cfg = self._urbits[patp]
        svc = data['svc_type']
        alias = data['alias']
        op = data['operation']

        # Urbit URL
        if svc == 'urbit-web':
            if op == 'create':
                print(f"{patp}: Attempting to register custom domain for {svc}")
                if self.dns_record(patp, cfg['wg_url'], alias):
                    if self.wg.handle_alias(patp, alias, 'post'):
                        self._urbits[patp]['custom_urbit_web'] = alias
                        self._urbits[patp]['show_urbit_web'] = 'alias'
                        if self.save_config(patp):
                            return 200
            elif op == 'delete':
                print(f"{patp}: Attempting to delete custom domain for {svc}")
                if self.wg.handle_alias(patp, alias, 'delete'):
                    self._urbits[patp]['custom_urbit_web'] = ''
                    self._urbits[patp]['show_urbit_web'] = 'default'
                    if self.save_config(patp):
                        return 200

        # MinIO URL
        if svc == 'minio':
            if op == 'create':
                print(f"{patp}: Attempting to register custom domain for {svc}")
                if self.dns_record(patp, f"s3.{cfg['wg_url']}", alias):
                    if self.wg.handle_alias(f"s3.{patp}", alias, 'post'):
                        self._urbits[patp]['custom_s3_web'] = alias
                        if self.save_config(patp):
                            return 200

            elif op == 'delete':
                print(f"{patp}: Attempting to delete custom domain for {svc}")
                if self.wg.handle_alias(f"s3.{patp}", alias, 'delete'):
                    self._urbits[patp]['custom_s3_web'] = ''
                    if self.save_config(patp):
                        return 200
        return 400

    def dns_record(self, patp, real, mask):
        count = 0
        while count < 3:
            print(f"{patp}: Checking DNS records")
            ori = False
            alias = False
            try:
                ori = socket.getaddrinfo(real, None, socket.AF_INET, socket.SOCK_STREAM)[0][4][0]
                print(f"{patp}: {real} is {ori}")
            except:
                print(f"{patp}: {real} has no record")

            try:
                alias = socket.getaddrinfo(mask, None, socket.AF_INET, socket.SOCK_STREAM)[0][4][0]
                print(f"{patp}: {mask} is {alias}")
            except:
                print(f"{patp}: {mask} has no record")

            if ori and alias:
                if ori == alias:
                    print(f"{patp}: DNS records match")
                    return True

            count += 1
            time = count * 2
            print(f"{patp}: Checking DNS record again in {time} seconds")
            sleep(time)

        print(f"{patp}: DNS records do not match or does not exist")
        return False

    # Swap Display Url
    def swap_url(self, patp):
        try:
            old = self._urbits[patp]['show_urbit_web']

            if old == 'alias':
                self._urbits[patp]['show_urbit_web'] = 'default'
            else:
                self._urbits[patp]['show_urbit_web'] = 'alias'

            print(f"{patp}: Urbit web display URL changed: {old} -> {self._urbits[patp]['show_urbit_web']}")
            self.save_config(patp)
            return 200
        except Exception as e:
            print(f"{patp}: Failed to change urbit web display URL: {e}")
        return 400


    # Container logs
    def logs(self, patp,timestamps=False):
        return self.urb_docker.full_logs(patp,timestamps)

    '''






    def load_config(self, patp):
        try:
            with open(f"{self.cfg.base}/settings/pier/{patp}.json") as f:
                cfg = json.load(f)
                self._urbits[patp] = {**default_pier_config, **cfg}

                # Updater Urbit information
                try:
                    if (self.cfg.version_server_ready) and (self.cfg.system.get('updateMode') != 'off'):
                        print(f"groundseg:urbit:{patp}:load_config: Replacing local data with version server data")
                        self._urbits[patp]['urbit_repo'] = self.version_info['repo']
                        self._urbits[patp]['urbit_version'] = self.version_info['tag']
                        self._urbits[patp]['urbit_amd64_sha256'] = self.version_info['amd64_sha256']
                        self._urbits[patp]['urbit_arm64_sha256'] = self.version_info['arm64_sha256']
                        self._urbits[patp]['minio_repo'] = self.version_minio['repo']
                        self._urbits[patp]['minio_version'] = self.version_minio['tag']
                        self._urbits[patp]['minio_amd64_sha256'] = self.version_minio['amd64_sha256']
                        self._urbits[patp]['minio_arm64_sha256'] = self.version_minio['arm64_sha256']
                        self.save_config(patp)
                except Exception as e:
                    pass

                print(f"groundseg:urbit:{patp}:load_config: Config loaded")
                return True
        except Exception as e:
            print(f"groundseg:urbit:{patp}:load_config: Failed to load config: {e}")
            return False

    def save_config(self, patp, dupe=None): # dupe is a temporary fix for the updater loop
        try:
            with open(f"{self.cfg.base}/settings/pier/{patp}.json", "w") as f:
                if dupe:
                    self._urbits[patp] = dupe
                json.dump(self._urbits[patp], f, indent = 4)
                print(f"groundseg:urbit:{patp}:save_config: Config saved")
                return True
        except Exception as e:
            print(f"groundseg:urbit:{patp}:save_config: Failed to save config: {e}")
            return False
