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
            data = {
                    "reg_code":self.ws_util.grab_form(sid,'startram','key'),
                    "pubkey":self.config['pubkey'],
                    "region":self.ws_util.grab_form(sid,'startram','region') or "us-east"
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
        tries = 0
        pubkey = f"pubkey={self.config['pubkey']}"
        while True:
            try:
                Log.log(f"{self._f}:retrieve_status Attempting to retrieve information")
                status = requests.post(f"{self.url}/retrieve?{pubkey}",
                                       headers=self._headers
                                       ).json()
                '''
                Log.log(f"{self.url}/retrieve?{pubkey}")
                Log.log(status)
                return True
                '''
                Log.log(f"{self._f}:retrieve_status Response: {status}")
                if status and status['conf']:
                    self.wg.anchor_data = status
                    Log.log(f"{self._f}:retrieve_status Response: {status}")
                    return True
                else:
                    raise Exception()

            except Exception as e:
                Log.log(f"{self._f}:retrieve_status Failed to retrieve status from '{self.url}/retrieve': {e}")
                if tries >= max_tries:
                    Log.log(f"{self._f}:retrieve_status Max retries exceeded ({max_tries})")
                    return False

            Log.log(f"{self._f}:retrieve_status Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1

    # /v1/create
    def register_service(self, subdomain, service_type, max_tries=1):
        data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        tries = 0
        while True:
            try:
                res = requests.post(f"{self.url}/create",json=data,headers=self._headers).json()
                Log.log(f"{self._f}:register_service:{subdomain} Sent service creation request")
            except Exception as e:
                Log.log(f"{self._f}:register_service:{subdomain} Failed to register service {service_type}: {e}")
                if tries >= max_tries:
                    Log.log(f"{self._f}:register_service:{subdomain} Max retries exceeded ({max_tries})")
                    return
            Log.log(f"{self._f}:register_service:{subdomain} Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1
        
        # check response for creating

        '''
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
        '''
