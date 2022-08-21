import json, subprocess, requests
from wireguard import Wireguard
from urbit_docker import UrbitDocker

class Orchestrator:
    
    _urbits = {}


    def __init__(self, config_file):

        self.config_file = config_file
        # Load config
        with open(config_file) as f:
            self.config = json.load(f)

        # First boot setup
        if(self.config['firstBoot']):
            self.first_boot()
            self.config['firstBoot'] = False
            self.save_config()

        self.wireguard = Wireguard(self.config)
        self.load_urbits()

    def load_urbits(self):
        for p in self.config['piers']:
            data = None
            with open(f'settings/{p}.json') as f:
                data = json.load(f)

            self._urbits[p] = UrbitDocker(data)

    def addUrbit(self, patp, urbit):
        self.config['piers'].append(patp)
        self._urbits[patp] = urbit
        self.save_config()

    def removeUrbit(self, patp):
        urb = self._urbits[patp]
        urb.removeUrbit()
        urb = self._urbits.pop(patp)
        self.config['piers'].remove(patp)
        self.save_config()


    def getUrbits(self):
        urbits= []

        for urbit in self._urbits.values():
            u = dict()
            u['name'] = urbit.pier_name
            u['running'] = urbit.isRunning();
            u['url'] = f'http;//192.168.0.229:{urbit.config["http_port"]}'
            if(urbit.isRunning()):
                u['code'] = urbit.get_code().decode('utf-8')
            else:
                u['code'] = ""
            urbits.append(u)
        return urbits
    
    def getContainers(self):
        containers = list(self._urbits.keys())
        containers.append('wireguard')
        containers.append('minio')
        return containers

    def getLogs(self, container):
        if container == 'wireguard':
            return self.wireguard.wg_docker.logs()
        if container == 'minio':
            return "" #TODO add minio container to orch
        if container in self._urbits.keys():
            return self._urbits[container].logs()
        return ""


    def first_boot(self):
        subprocess.run("wg genkey > privkey", shell=True)
        subprocess.run("cat privkey| wg pubkey | base64 -w 0 > pubkey", shell=True)

        # Load priv and pub key
        with open('pubkey') as f:
           self.config['pubkey'] = f.read().strip()
        with open('privkey') as f:
           self.config['privkey'] = f.read().strip()
        #clean up files
        #subprocess.run("rm privkey pubkey", shell =True)
   

    def save_config(self):
        with open(self.config_file, 'w') as f:
            json.dump(self.config, f, indent = 4)




if __name__ == '__main__':
    orchestrator = Orchestrator("settings/system.json")

