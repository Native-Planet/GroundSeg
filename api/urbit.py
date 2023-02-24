# Python
import copy
import json
import shutil

# GroundSeg Modules
from log import Log
from utils import Utils
from urbit_docker import UrbitDocker

default_pier_config = {
        "pier_name":"",
        "http_port":8080,
        "ames_port":34343,
        "loom_size":31,
        "urbit_version":"latest",
        "minio_version":"latest",
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
        "boot_status": "boot"
        }


class Urbit:

    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.urb_docker = UrbitDocker()
        self._urbits = {}

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("Urbit: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater Wireguard information
        if self.config_object.update_avail:
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['vere']

        self.start_all(self.config['piers'])

    # Start container
    def start(self, patp):
        if self.load_config(patp):
            return self.urb_docker.start(patp, self.updater_info)
        else:
            return "failed"

    # Start all valid containers
    def start_all(self, patps):
        Log.log("Urbit: Starting all ships")
        res = {"failed":[],"succeeded":[],"ignored":[],"invalid":[]}
        if len(patps) < 1:
            Log.log(f"Urbit: No ships detected in system.json! Skipping..")
            return True

        for p in patps:
            status = self.start(p)
            try:
                res[status].append(p)
            except Exception as e:
                Log.log(f"{p}: {e}")

        Log.log(f"Urbit: Start succeeded {res['succeeded']}")
        Log.log(f"Urbit: Start ignored {res['ignored']}")
        Log.log(f"Urbit: Start failed {res['failed']}")
        Log.log(f"Urbit: Patp invalid {res['invalid']}")

        return True

    # Return list of ship information
    def list_ships(self):
        urbits = []
        try:
            if len(self.config['piers']) > 0:
                for patp in self.config['piers']:
                    try:
                        u = dict()
                        c = self.urb_docker.get_container(patp)
                        if c:
                            u['name'] = patp
                            u['running'] = c.status == "running"
                            # TODO: Urbit Config
                            #u['url'] = f'http://{socket.gethostname()}.local:{urbit.config["http_port"]}'

                        #if(urbit.config['network']=='wireguard'):
                        #    u['url'] = f"https://{urbit.config['wg_url']}"

                            urbits.append(u)
                    except Exception as e:
                        Log.log(f"{patp}: {e}")
        except Exception as e:
            Log.log(f"Urbit: Unable to list Urbit ships: {e}")

        return {'urbits': urbits}

    # Boot new pier from key
    def create(self, patp, key):
        Log.log(f"{patp}: Attempting to boot new urbit ship")
        try:
            if not Utils.check_patp(patp):
                raise Exception("Invalid @p")

            # todo: Add check if exists, return prompt to user for further action
            
            # Get open ports
            http_port, ames_port = self.get_open_urbit_ports()

            # Generate config file for pier
            cfg = self.build_config(patp, http_port, ames_port)
            self._urbits[patp] = cfg
            self.save_config(patp)

            # Delete old volume
            try:
                # TODO: remove with docker
                shutil.rmtree(f"{self._volume_directory}/{patp}")
                Log.log(f"{patp}: Removed volume")
            except:
                Log.log(f"{patp}: No old volume to delete")

                # Create the docker container
                if self.urb_docker.create(cfg, self.updater_info, key, self.config_object._arch, self._volume_directory):
                    Log.log(f"{patp}: Adding to system.json")

                    self.config['piers'].append(patp)
                    self.config_object.save_config()

                    # startram stuff
                    #self.add_urbit(patp, urbit)

                    if self.start(patp) == "suceeded":
                        return 200

        except Exception as e:
            Log.log(f"{patp}: Failed to boot new urbit ship: {e}")

        return 400

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

    def load_config(self, patp):
        try:
            with open(f"{self.config_object.base_path}/settings/pier/{patp}.json") as f:
                self._urbits[patp] = json.load(f)
                Log.log(f"{patp}: Config loaded")
                return True
        except Exception as e:
            Log.log(f"{patp}: Failed to load config: {e}")
            return False

    def save_config(self, patp):
        try:
            with open(f"{self.config_object.base_path}/settings/pier/{patp}.json", "w") as f:
                json.dump(self._urbits[patp], f, indent = 4)
                Log.log(f"{patp}: Config saved")
                return True
        except Exception as e:
            Log.log(f"{patp}: Failed to save config: {e}")
            return False
    '''
    # Stop container
    def stop(self):
        return self.wg_docker.stop(self.data)

    def remove(self):
        return self.wg_docker.remove_wireguard(self.data['wireguard_name'])

    # Is container running
    def is_running(self):
        return self.wg_docker.is_running(self.data['wireguard_name'])

    # Container logs
    def logs(self):
        return self.wg_docker.logs(self.data['wireguard_name'])
'''
