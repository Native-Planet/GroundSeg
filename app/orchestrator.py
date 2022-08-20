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

    def getUrbits(self):
        running =[]
        stopped = []
        for urbit in self._urbits:
            print(self._urbits[urbit].container.attrs['State']['Status'])

            if self._urbits[urbit].isRunning():
                running.append(urbit)
            else:
                stopped.append(urbit)

        return running,stopped


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

