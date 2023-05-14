import requests
from time import sleep

from log import Log

class StartramAPI:
    _f = "startram_api"
    _headers = {"Content-Type": "application/json"}
    def __init__(self, config, wg, ws_util):
        self.config_object = config
        self.config = config.config
        self.url = f"https://{self.config['endpointUrl']}/{self.config['apiVersion']}"
        self.wg = wg
        self.ws_util = ws_util

    # /v1/register
    def register_device(self,sid):
        Log.log(f"{self._f}:register_device Attempting to register device")
        try:
            region = self.ws_util.grab_form(sid,'startram','region')
            if region == None:
                region = "us-east"

            reg_code = self.ws_util.grab_form(sid,'startram','key')
            region = self.ws_util.grab_form(sid,'startram','region') or "us-east"
            data = {
                    "reg_code":reg_code,
                    "pubkey":self.config['pubkey'],
                    "region":region
                    }

            res = requests.post(f"{self.url}/register",
                                json=data,
                                headers=self._headers
                                ).json()

            Log.log(f"{self._f}:register_device /register response: {res}")
            if res['error'] != 0:
                raise Exception(f"error not 0: {res}")
            return True

        except Exception as e:
            Log.log(f"{self._f}:register_device Request to /register failed: {e}")
        return False

    # /v1/retrieve
    def retrieve_status(self, max_tries=1):
        tries = 1
        pubkey = f"pubkey={self.config['pubkey']}"
        url = f"{self.url}/retrieve?{pubkey}"

        while True:
            try:
                Log.log(f"{self._f}:retrieve_status Attempting to retrieve information")
                status = requests.get(url, headers=self._headers).json()
                Log.log(f"{self._f}:retrieve_status Response: {status}")

                if status and status['conf']:
                    self.wg.anchor_data = status
                    Log.log(f"{self._f}:retrieve_status Response: {status}")
                    return True
                else:
                    raise Exception()

            except Exception as e:
                Log.log(f"{self._f}:retrieve_status Failed to retrieve status from '{url}': {e}")
                if tries >= max_tries:
                    Log.log(f"{self._f}:retrieve_status Max retries exceeded ({max_tries})")
                    return False

            Log.log(f"{self._f}:retrieve_status Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1

    # /v1/create
    def create_service(self, subdomain, service_type, max_tries=1):
        data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        patp = subdomain
        if 's3' in subdomain:
            patp = subdomain.split('.')[-1]
        tries = 0
        while True:
            try:
                res = requests.post(f"{self.url}/create",json=data,headers=self._headers)
                Log.log(f"{self._f}:create_service:{subdomain} Sent service creation request")
                if res.status_code == 200:
                    if "s3" in subdomain:
                        self.ws_util.urbit_broadcast(patp, 'startram', 'minio', 'registering')
                    else:
                        self.ws_util.urbit_broadcast(patp, 'startram', 'urbit', 'registering')
                    break
                else:
                    raise Exception(f"status code: {res.status_code}, res: {res.json()}")

                    #self.ws_util.urbit_broadcast(patp, 'startram', 'access', 'unregistered') # remote, local
            except Exception as e:
                Log.log(f"{self._f}:create_service:{subdomain} Failed to register service {service_type}: {e}")
                if tries >= max_tries:
                    Log.log(f"{self._f}:create_service:{subdomain} Max retries exceeded ({max_tries})")
                    break
            Log.log(f"{self._f}:create_service:{subdomain} Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1

    # /v1/delete
    def delete_service(self, subdomain, service_type):
        data = {
                "subdomain" : f"{subdomain}",
                "pubkey":self.config['pubkey'],
                "svc_type": service_type
                }
        try:
            response = requests.post(f'{self.url}/delete',json=data,headers=self._headers).json()
            Log.log(f"startram_api:delete_service Service {service_type} deleted: {response}")
        except Exception:
            Log.log(f"startram_api:delete_service Failed to delete service {service_type}")
        
    # /v1/regions
    def get_regions(self, max_tries=1):
        Log.log(f"{self._f}:get_regions Attempting to get regions")
        tries = 1
        while True:
            try:
                self.wg.region_data = requests.get(
                        f"{self.url}/regions",
                        headers=self._headers
                        ).json()
                return True
            except Exception as e:
                Log.log(f"{self._f}:get_regions Failed: {e}")
                if tries >= max_tries:
                    Log.log(f"{self._f}:get_regions Max retries exceeded ({max_tries})")
                    self.wg.region_data = {}
                    break
            Log.log(f"{self._f}:get_regions Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1
        return False

    # /v1/stripe/cancel
    def cancel_subscription(self, key):
        try:
            Log.log(f"{self._f}:cancel_subscription Attempting to cancel StarTram subscription")
            data = {'reg_code': reg_key}
            res = requests.post(f'{self.url}/stripe/cancel',json=data,headers=headers).json()
            if res['error'] == 0:
                if self.retrieve_status():
                    Log.log(f"{self._f}:cancel_subscription StarTram subscription canceled")
                    return True
                else:
                    raise Exception(f"non-zero error code: {res}")
        except Exception as e:
            Log.log(f"{self._f}:cancel_subscription Failed: {e}")
        return False
