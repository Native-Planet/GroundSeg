import requests, subprocess, base64, time, json

from wireguard_docker import WireguardDocker

class Wireguard:
    _url = "https://api1.nativeplanet.live/v1"


    _headers = {"Content-Type": "application/json"}

    _service_type = ['minio','urbit-web','urbit-ames']



    def __init__(self, config):
        self.config = config

        # load wireguard docker
        filename = "settings/wireguard.json"
        f = open(filename)
        data = json.load(f)
        self.wg_docker = WireguardDocker(data)
        if(self.wg_docker.isRunning()):
            self.wg_docker.stop()
        

    def start(self):
        self.wg_docker.start()


    def registerDevice(self, reg_code):
        # /v1/register
        update_data = {
            "reg_code" : f"{reg_code}",
            "pubkey":self.config['pubkey']
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{self._url}/register',json=update_data,headers=headers).json()
        except Exception as e:
            print(e)
            return None

        return(response['lease'])

    def registerService(self, subdomain, service_type):
        # /v1/create
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{self._url}/create',json=update_data,headers=headers).json()
        except Exception as e:
            print(e)
            return None
        
        # wait for it to be created
        while response['status'] == 'creating':
            try:
                response = requests.get(
                        f'{self._url}/retrieve?pubkey={update_data["pubkey"]}',
                        headers=headers).json()
            except Exception as e:
                print(e)
            print("Waiting for endpoint to be created")
            if(response['status'] == 'creating'):
                time.sleep(60)

        return response['status']
        
    def getStatus(self):
        headers = {"Content-Type": "application/json"}
        response = None
        try:
            response = requests.get(
                    f'{self._url}/retrieve?pubkey={self.config["pubkey"]}',
                    headers=headers).json()
        except Exception as e:
            print(e)

        self.wg_config = base64.b64decode(response['conf']).decode('utf-8')

        self.wg_config = self.wg_config.replace('privkey', self.config['privkey'])
        # Setup and start the local wg client
        self.wg_docker.addConfig(self.wg_config)
        return response
 

    def setupWireguard(self, patp):
        # Setup the wireguard server
        response = self.check_wireguard();
        if response['error']==1:
            self.wg_config = self.create_wireguard(patp).decode('utf-8')
        else:
            print(response)
            self.wg_config = base64.b64decode(response['conf']).decode('utf-8')

        self.wg_config = self.wg_config.replace('privkey', self.config['privkey'])
        # Setup and start the local wg client
        self.wg_docker.addConfig(self.wg_config)

