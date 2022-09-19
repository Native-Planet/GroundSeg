import requests, subprocess, base64, time, json

from wireguard_docker import WireguardDocker

class Wireguard:

    _headers = {"Content-Type": "application/json"}


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
    def stop(self):
        self.wg_docker.stop()


    def registerDevice(self, reg_code, url):
        # /v1/register
        update_data = {
            "reg_code" : f"{reg_code}",
            "pubkey":self.config['pubkey']
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{url}/register',json=update_data,headers=headers).json()
        except Exception as e:
            print(f"register device {e}")
            return None

        return(response['lease'])

    def registerService(self, subdomain, service_type, url):
        # /v1/create
        update_data = {
            "subdomain" : f"{subdomain}",
            "pubkey":self.config['pubkey'],
            "svc_type": service_type
        }
        headers = {"Content-Type": "application/json"}

        response = None
        try:
            response = requests.post(f'{url}/create',json=update_data,headers=headers).json()
        except Exception as e:
            print(e)
            return None
        
        # wait for it to be created
        while response['status'] == 'creating':
            try:
                response = requests.get(
                        f'{url}/retrieve?pubkey={update_data["pubkey"]}',
                        headers=headers).json()
            except Exception as e:
                print(e)
            print("Waiting for endpoint to be created")
            if(response['status'] == 'creating'):
                time.sleep(60)

        return response['status']
        
    def getStatus(self,url):
        headers = {"Content-Type": "application/json"}
        response = None

        try:
            response = requests.get(
                    f'{url}/retrieve?pubkey={self.config["pubkey"]}',
                    headers=headers).json()
        except Exception as e:
            print(e)

        count =0
        while ((response['conf'] == None) and (count > 6)):
            try:
                response = requests.get(
                        f'{url}/retrieve?pubkey={self.config["pubkey"]}',
                        headers=headers).json()
            except Exception as e:
                print(e)
            count = count +1

        try:
            self.wg_config = base64.b64decode(response['conf']).decode('utf-8')

            self.wg_config = self.wg_config.replace('privkey', self.config['privkey'])
            # Setup and start the local wg client
            self.wg_docker.addConfig(self.wg_config)
        except Exception as e:
            print(response)
            print(f"get Status {e}")
            return None

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

    def isRunning(self):
        return self.wg_docker.isRunning()

