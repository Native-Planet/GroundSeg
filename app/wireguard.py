import requests, subprocess, base64, time, json

from wireguard_docker import WireguardDocker

class Wireguard:
    _url = "https://api1.nativeplanet.live/v1"


    _headers = {"Content-Type": "application/json"}



    def __init__(self, config):
        self.config = config

        # load wireguard docker
        filename = "settings/wireguard.json"
        f = open(filename)
        data = json.load(f)
        self.wg_docker = WireguardDocker(data)
        

    def start(self):
        self.wg_docker.start()

    def setupWireguard(self, patp):
        # Setup the wireguard server
        response = self.check_wireguard();
        if response['error']==1:
            self.wg_config = self.create_wireguard(patp)
        else:
            self.wg_config = base64.b64decode(response['conf'])

        # Setup and start the local wg client
        self.wg_docker.addConfig(self.wg_config)



    def create_wireguard(self,patp):
        update_data = {
            "patp" : f"{patp}",
            "pubkey":self.config['pubkey']
        }

        response = None
        try:
            response = requests.post(f'{url}/create', json = update_data, headers=headers).json()
        except Exception as e:
            print(e)
            return None

        # wait for it to be created
        while response['status'] != 'ready':
            try:
                response = requests.get(f'{url}/retrieve?pubkey={pubkey}', headers=headers).json()
            except Exception as e:
                print(e)
                return None
            print("Waiting for endpoint to be created")
        time.sleep(60)

        return base64.b64decode(response['conf'])


    def check_wireguard(self):
        response = None
        try:
            response = requests.get(f'{self._url}/retrieve?pubkey={self.config["pubkey"]}',
                                    headers=headers).json()
        except Exception as e:
            print(e)

        return response

