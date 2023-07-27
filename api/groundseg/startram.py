import asyncio
import requests
from time import sleep

#from log import Log

class StarTramAPI:
    _f = "groundseg:startram"
    _headers = {"Content-Type": "application/json"}
    def __init__(self,cfg,wg,urbit):
        self.cfg = cfg
        self.wg = wg
        self.urbit = urbit

    async def main_loop(self):
        retrieve_sleep = 60 * 60 #seconds
        retrieve_count = 0

        region_sleep = 60 * 2 #seconds
        region_count = 0

        while True:
            self.url = self.cfg.system.get('endpointUrl')
            try:
                if self.cfg.system.get('wgRegistered') and (retrieve_count == 0 or retrieve_count % retrieve_sleep == 0):
                    if not self.retrieve_status(30):
                        raise Exception("retrieve status returned False")
                    retrieve_count = retrieve_count + 1

                if self.cfg.system.get('setup') == "startram" and (region_count == 0 or region_count % region_sleep == 0):
                    if not self.get_regions(10):
                        raise Exception("get regions returned False")
                    region_count = region_count + 1
            except Exception as e:
                print(f"groundseg:startram: {e}")
            await asyncio.sleep(1)

    # /v1/register
    def register_device(self, reg_code, region="us-east"):
        print(f"{self._f}:register_device Attempting to register device")
        try:
            pubkey = self.cfg.system.get('pubkey'),
            data = {
                    "reg_code":reg_code,
                    "pubkey":pubkey[0],
                    "region":region
                    }
            res = requests.post(f"https://{self.url}/v1/register",
                                json=data,
                                headers=self._headers
                                ).json()
            print(f"{self._f}:register_device /register response: {res}")
            if res['error'] != 0:
                raise Exception(f"error not 0: {res}")
            self.cfg.system['wgRegistered'] = True
            self.cfg.save_config()
            return True
        except Exception as e:
            print(f"{self._f}:register_device Request failed: {e}")
        return False

    # /v1/retrieve
    def retrieve_status(self, max_tries=1):
        tries = 1
        pubkey = f"pubkey={self.cfg.system.get('pubkey')}"
        url = f"https://{self.url}/v1/retrieve?{pubkey}"
        while True:
            try:
                print(f"{self._f}:retrieve_status Attempting to retrieve information")
                status = requests.get(url, headers=self._headers).json()
                #print(f"{self._f}:retrieve_status Response: {status}")

                if status.get('status') == "No record":
                    self.cfg.system['wgRegistered'] = False
                    self.cfg.save_config()
                    return True

                if status and status['conf']:
                    self.wg.anchor_data = status
                    if self.wg.get_subdomains():
                        self.urbit.update_wireguard_network()
                    return True
                else:
                    raise Exception()

            except Exception as e:
                print(f"{self._f}:retrieve_status Failed to retrieve status from '{url}': {e}")
                if tries >= max_tries:
                    print(f"{self._f}:retrieve_status Max retries exceeded ({max_tries})")
                    return False

            print(f"{self._f}:retrieve_status Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1

    # /v1/create
    def create_service(self, subdomain, service_type, max_tries=1):
        pubkey = self.cfg.system.get('pubkey')
        data = {
            "subdomain": f"{subdomain}",
            "pubkey": pubkey,
            "svc_type": service_type
        }
        patp = subdomain
        if 's3' in subdomain:
            patp = subdomain.split('.')[-1]
        tries = 0
        while True:
            try:
                res = requests.post(f"https://{self.url}/v1/create",json=data,headers=self._headers)
                print(f"{self._f}:create_service:{subdomain} Sent service creation request")
                if res.status_code == 200:
                    break
                else:
                    raise Exception(f"status code: {res.status_code}, res: {res.json()}")
            except Exception as e:
                print(f"{self._f}:create_service:{subdomain} Failed to register service {service_type}: {e}")
                if tries >= max_tries:
                    print(f"{self._f}:create_service:{subdomain} Max retries exceeded ({max_tries})")
                    break
            print(f"{self._f}:create_service:{subdomain} Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1

    def unregister(self):
        # unregister all services
        # update config
        self.cfg.set_wg_registered(False)
        return True

    # /v1/delete
    def delete_service(self, subdomain, service_type):
        pubkey = self.cfg.system.get('pubkey')
        data = {
                "subdomain": f"{subdomain}",
                "pubkey":pubkey,
                "svc_type": service_type
                }
        try:
            response = requests.post(f'https://{self.url}/v1/delete',json=data,headers=self._headers).json()
            print(f"api:startram:delete_service Service {service_type} deleted: {response}")
        except Exception:
            print(f"api:startram:delete_service Failed to delete service {service_type}")

    # /v1/regions
    def get_regions(self, max_tries=1):
        print(f"{self._f}:get_regions Attempting to get regions")
        tries = 1
        while True:
            try:
                self.wg.region_data = requests.get(
                        f"https://{self.url}/v1/regions",
                        headers=self._headers
                        ).json()
                return True
            except Exception as e:
                print(f"{self._f}:get_regions Failed: {e}")
                if tries >= max_tries:
                    print(f"{self._f}:get_regions Max retries exceeded ({max_tries})")
                    self.wg.region_data = {}
                    break
            print(f"{self._f}:get_regions Retrying in {tries * 2} seconds")
            sleep(tries * 2)
            tries += 1
        return False

    # /v1/stripe/cancel
    def cancel_subscription(self, key):
        try:
            print(f"{self._f}:cancel_subscription Attempting to cancel StarTram subscription")
            data = {'reg_code': key}
            res = requests.post(f'https://{self.url}/v1/stripe/cancel',json=data,headers=self._headers).json()
            if res['error'] == 0:
                if self.retrieve_status():
                    print(f"{self._f}:cancel_subscription StarTram subscription canceled")
                    return True
                else:
                    raise Exception(f"non-zero error code: {res}")
        except Exception as e:
            print(f"{self._f}:cancel_subscription Failed: {e}")
        return False
