import sys
import json
import base64
import requests
import subprocess
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
        self.anchor_data = {}
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
        self.data = {**self.default_config, **self.data}

        self.save_config()

        if self.config['wgOn'] and self.config['wgRegistered']:
            self.start()

        Log.log("Wireguard: Initialization Completed")

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

    # wgOn False
    def off(self, urb, minio):
        for p in urb._urbits:
            if urb._urbits[p]['network'] == 'wireguard':
                 urb.toggle_network(p)
        minio.stop_all()
        minio.stop_mc()
        self.stop()
        self.config['wgOn'] = False
        self.config_object.save_config()

        return 200

    # wgOn False
    def on(self, minio):
        self.start()
        minio.start_mc()
        minio.start_all()
        self.config['wgOn'] = True
        self.config_object.save_config()

        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f'https://{endpoint}/{api_version}'

        if self.get_status(url):
            self.update_wg_config(self.anchor_data['conf'])

        return 200

    def restart(self, urb, minio):
        try:
            Log.log("Wireguard: Attempting to restart wireguard")
            self.config_object.anchor_ready = False
            Log.log("Anchor: Refresh loop is unready")
            remote = []
            for patp in urb._urbits:
                if urb._urbits[patp]['network'] != 'none':
                    remote.append(patp)

            if self.off(urb, minio) == 200:
                if self.on(minio) == 200:
                    if len(remote) <= 0:
                        return 200
                    for patp in remote:
                        if urb.toggle_network(patp) == 200:
                            Log.log("Anchor: Refresh loop is ready")
                            self.config_object.anchor_ready = True
                            return 200
        except Exception as e:
            Log.log(f"Wireguard: Failed to restart wireguard: {e}")

        return 400

    # Container logs
    def logs(self):
        return self.wg_docker.full_logs(self.data['wireguard_name'])

    # New anchor registration
    def build_anchor(self, url, reg_key):
        Log.log("Wireguard: Attempting to build anchor")
        try:
            if self.register_device(url, reg_key):
                if self.get_status(url):
                    if self.start():
                        if self.update_wg_config(self.anchor_data['conf']):
                            Log.log("Anchor: Registered with anchor server")
                            return True

        except Exception as e:
            Log.log(f"Wireguard: Failed to build anchor: {e}")

        return False

    # Update wg0.confg
    def update_wg_config(self, conf):
        try:
            conf = base64.b64decode(conf).decode('utf-8')
            conf = conf.replace('privkey', self.config['privkey'])
            return self.wg_docker.add_config(self.data, conf)

        except Exception as e:
            Log.log(f"Wireguard: Failed to update wg0.confg: {e}")

    # Change Anchor endpoint URL
    def change_url(self, url, urb, minio):
        Log.log(f"Wireguard: Attempting to change endopint url to {url}")
        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        old_url = f'https://{endpoint}/{api_version}'
        self.config['endpointUrl'] = url
        self.config['wgRegistered'] = False
        self.config['wgOn'] = False

        for patp in self.config['piers']:
            self.delete_service(f'{patp}','urbit',old_url)
            self.delete_service(f's3.{patp}','minio',old_url)

        self.off(urb, minio)
        self.config_object.reset_pubkey()
        Log.log("Wireguard: Changed url")
        self.config_object.save_config()
        if self.config['endpointUrl'] == url:
            return 200
        return 400

    # Container logs
    def logs(self, name):
        return self.wg_docker.full_logs(name)

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
#   StarTram API
#

    # /v1/register
    def register_device(self, url, reg_key):
        Log.log("Anchor: Attempting to register device")
        try:
            update_data = {"reg_code" : f"{reg_key}","pubkey":self.config['pubkey']}
            response = None

            res = requests.post(f'{url}/register',json=update_data,headers=self._headers).json()
            Log.log(f"Anchor: /register response: {res}")
            if res['error'] != 0:
                raise Exception("error not 0")

            return True

        except Exception as e:
            Log.log(f"Anchor: Request to /register failed: {e}")

        return False

    # /v1/retrieve
    def get_status(self, url):
        full_url = f"{url}/retrieve?pubkey={self.config['pubkey']}"
        err_count = 0
        while err_count < 6:
            try:
                self.anchor_data = requests.get(full_url,headers=self._headers).json()
                return True

            except Exception as e:
                Log.log(f"Anchor: /retrieve failed: {e}")
                err_count = err_count + 1

        return False

    # /v1/create
    def register_service(self, subdomain, service_type, url):
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        response = False
        while not response:
            try:
                response = requests.post(f'{url}/create',json=update_data,headers=headers).json()
                Log.log(f"Anchor: Sent creation request for {service_type}")
            except Exception as e:
                Log.log(f"Anchor: Failed to register service {service_type}: {e}")
        
        # wait for it to be created
        while response['status'] == 'creating':
            try:
                response = requests.get(
                        f'{url}/retrieve?pubkey={update_data["pubkey"]}',
                        headers=headers).json()
                Log.log(f"Anchor: Retrieving response for {service_type}")
            except Exception as e:
                Log.log(f"Anchor: Failed to retrieve response: {e}")

            if(response['status'] == 'creating'):
                Log.log("Anchor: Waiting for endpoint to be created")
                sleep(60)

        return response['status']

    # /v1/create/alias
    def handle_alias(self, patp, alias, req_type):
        endpoint = self.config['endpointUrl']
        api_version = self.config['apiVersion']
        url = f"https://{endpoint}/{api_version}"

        headers = {"Content-Type": "application/json"}

        blob = {
            "subdomain": patp,
            "alias": alias,
            "pubkey": self.config['pubkey']
        }
        if req_type == 'post':
            try:
                response = requests.post(f'{url}/create/alias',json=blob,headers=headers).json()
                Log.log(f"Anchor: Sent alias {alias} creation request for {patp}")
                Log.log(f"Anchor: {response}")
                if response['error'] == 0:
                    return True
            except Exception as e:
                Log.log(f"Anchor: Failed to register alias {alias} for {patp}: {e}")

        elif req_type == 'delete':
            try:
                response = requests.delete(f'{url}/create/alias',json=blob,headers=headers).json()
                Log.log(f"Anchor: Sent alias {alias} deletion request for {patp}")
                Log.log(f"Anchor: {response}")
                if response['error'] == 0:
                    return True
            except Exception as e:
                Log.log(f"Anchor: Failed to delete alias {alias} for {patp}: {e}")

        return False

    # /v1/delete
    def delete_service(self, subdomain, service_type, url):
        Log.log(f"Anchor: Attempting to delete service {service_type}")
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        try:
            response = requests.post(f'{url}/delete',json=update_data,headers=headers).json()
            Log.log(f"Anchor: Service {service_type} deleted: {response}")
        except Exception as e:
            Log.log(f"Anchor: Failed to delete service {service_type}")
            return None
        
    # /v1/stripe/cancel
    def cancel_subscription(self, reg_key, url):
        Log.log(f"Anchor: Attempting to cancel subscription")
        headers = {"Content-Type": "application/json"}
        data = {'reg_code': reg_key}
        response = None

        try:
            response = requests.post(f'{url}/stripe/cancel',json=data,headers=headers).json()
            if response['error'] == 0:
                if self.get_status(url):
                    Log.log(f"Anchor: Successfully canceled subscription")
                    return 200

        except Exception as e:
            Log.log(f"Anchor: Cancelation failed: {e}")
            return 400
