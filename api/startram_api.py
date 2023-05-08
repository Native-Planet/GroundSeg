import requests

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
        from time import sleep
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

            Log.log(data)
            res = "fake data"
            '''
            res = requests.post(f"{self.url}/register",
                                json=update_data,
                                headers=self._headers
                                ).json()
            '''
            Log.log(f"{self._f}:register_device /register response: {res}")
            sleep(3)
            return True

            if res['error'] != 0:
                raise Exception(f"error not 0: {res}")
            return True

        except Exception as e:
            Log.log(f"{self._f}:register_device Request to /register failed: {e}")
        return False

    # /v1/retrieve
    def retrieve_status(self, max_tries=1):
        url = f"{self.url}/retrieve?pubkey={self.config['pubkey']}"
        tries = 0
        from time import sleep
        sleep(3)
        return True
        '''
        while True:
            try:
                status = requests.post(url, headers=self._headers).json()
                if status and status['conf']:
                    self.wg.anchor_data = status
                    return True
                else:
                    raise Exception()
            except:
                if tries >= max_tries:
                    return False
            sleep(count * 2)
            tries += 1

    # Register Wireguard for Urbit
    def register_urbit(self, patp, url):
        if self.config['wgRegistered']:
            Log.log(f"{patp}: Attempting to register anchor services")
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
                    Log.log(f"{patp}: Registering ship")
                    self.wg.register_service(f'{patp}', 'urbit', url)
 
                # One or more of the minio services is not registered
                if not (minio_svc and minio_console and minio_bucket):
                    Log.log(f"{patp}: Registering MinIO")
                    self.wg.register_service(f's3.{patp}', 'minio', url)

            svc_url = None
            http_port = None
            ames_port = None
            s3_port = None
            console_port = None
            tries = 1

            while None in [svc_url,http_port,ames_port,s3_port,console_port]:
                Log.log(f"{patp}: Checking anchor config if services are ready")
                if self.wg.get_status(url):
                    self.wg.update_wg_config(self.wg.anchor_data['conf'])

                Log.log(f"Anchor: {self.wg.anchor_data['subdomains']}")
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
                        Log.log(f"Anchor: {ep['svc_type']} not ready. Trying again in {t} seconds.")
                        time.sleep(t)
                        if tries <= 15:
                            tries = tries + 1
                        break

            return self.set_wireguard_network(patp, svc_url, http_port, ames_port, s3_port, console_port)
        return True

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
        '''
