import requests
import subprocess
import base64
import json
import sys

from time import sleep

# GroundSeg modules
from log import Log
from wireguard_docker import WireguardDocker

class Wireguard:

    _headers = {"Content-Type": "application/json"}
    data = {}
    updater_info = {}
    default_config = { 
                      "wireguard_name": "wireguard",
                      "wireguard_version": "latest",
                      "volume_dir": "/var/lib/docker/volumes",
                      "image": "linuxserver/wireguard",
                      "cap_add": ["NET_ADMIN","SYS_MODULE"],
                      "volumes": ["/lib/modules:/lib/modules"],
                      "sysctls": { "net.ipv4.conf.all.src_valid_mark": 1 }
                      }   

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.filename = f"{self.config_object.base_path}/settings/wireguard.json"
        self.wg_docker = WireguardDocker()

        self.load_config()

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("Wireguard: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater Wireguard information
        if self.config_object.update_avail:
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['wireguard']
            self.data['image'] = self.updater_info['repo']
            self.data['tag'] = self.updater_info['tag']
        self.data = {**self.default_config, **self.data}

        self.save_config()

        # TODO: if wgOn and wgRegistered
        if self.start():
            Log.log("Wireguard: Initialization Completed")

    # Load wireguard.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.data = json.load(f)
                Log.log("Wireguard: Successfully loaded wireguard.json")

        except Exception as e:
            Log.log(f"Wireguard: Failed to open wireguard.json: {e}")
            Log.log("Wireguard: New wireguard.json will be created")

    # Save wireguard.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.data, f, indent=4)
            f.close()


#
#   Wireguard Docker commands
#


    # Start container
    def start(self):
        return self.wg_docker.start(self.data, self.updater_info, self.config_object._arch)

    # Stop container
    def stop(self):
        return self.wg_docker.stop(self.data)

    # Remove container and volume
    def remove(self):
        return self.wg_docker.remove_wireguard(self.data['wireguard_name'])

    # Is container running
    def is_running(self):
        return self.wg_docker.is_running(self.data['wireguard_name'])

    # Container logs
    def logs(self):
        return self.wg_docker.logs(self.data['wireguard_name'])


#
#   StarTram endpoints
#

    # /v1/register
    def register_device(self, reg_code, url):
        update_data = {"reg_code" : f"{reg_code}","pubkey":self.config['pubkey']}
        response = None

        try:
            return requests.post(f'{url}/register',json=update_data,headers=self._headers).json()

        except Exception as e:
            print(f"/register failed: {e}", file=sys.stderr)
            return None

    # /v1/retrieve
    def get_status(self,url):
        response = None
        full_url = f'{url}/retrieve?pubkey={self.config["pubkey"]}'

        err_count = 0

        while err_count < 6:
            try:
                response = requests.get(full_url,headers=self._headers).json()
                break

            except Exception as e:
                print(f"/retrieve failed: {e}",file=sys.stderr)
                err_count = err_count + 1

        return response

    def update_wg_config(self, conf):
        try:
            self.wg_config = base64.b64decode(conf).decode('utf-8')
            self.wg_config = self.wg_config.replace('privkey', self.config['privkey'])
            self.wg_docker.add_config(self.data, self.wg_config)

        except Exception as e:
            print(f"wg_config err: {e}", file=sys.stderr)

    # /v1/create
    def register_service(self, subdomain, service_type, url):
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{url}/create',json=update_data,headers=headers).json()
            print(f"Sent creation request for {service_type}", file=sys.stderr)
        except Exception as e:
            print(e, file=sys.stderr)
            return None
        
        # wait for it to be created
        while response['status'] == 'creating':
            try:
                response = requests.get(
                        f'{url}/retrieve?pubkey={update_data["pubkey"]}',
                        headers=headers).json()
                print(f"Retrieving response for {service_type}", file=sys.stderr)
            except Exception as e:
                print(e)
            print("Waiting for endpoint to be created")
            if(response['status'] == 'creating'):
                sleep(60)

        return response['status']
        
    def delete_service(self, subdomain, service_type, url):
        # /v1/delete
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{url}/delete',json=update_data,headers=headers).json()
        except Exception as e:
            print(e)
            return None

        print(response, file=sys.stderr)
        
    def cancel_subscription(self, reg_key, url):
        # /v1/stripe/cancel
        headers = {"Content-Type": "application/json"}
        data = {'reg_code': reg_key}
        response = None

        try:
            response = requests.post(f'{url}/stripe/cancel',json=data,headers=headers).json()
            if response['error'] == 0:
                return self.get_status(url)

            print(response, file=sys.stderr)
            return 400

        except Exception as e:
            print(f'err: {e}', file=sys.stderr)
            return 400

