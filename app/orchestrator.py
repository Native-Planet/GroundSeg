import json, subprocess, requests
from wireguard import Wireguard
from urbit_docker import UrbitDocker
from minio_docker import MinIODocker
import time
import sys

class Orchestrator:
    
    _urbits = {}
    _minios = {}
    minIO_on = False


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

        # get wireguard netowkring information
        # Load urbits with wg info
        # start wireguard
        self.wireguard = Wireguard(self.config)
        if('reg_key' in self.config.keys()):
           if(self.config['reg_key']!= None):
              self.wireguard.start()

        self.load_urbits()


    def registerDevice(self, reg_key):
        self.config['reg_key'] = reg_key
        self.wireguard.registerDevice(self.config['reg_key']) 
        self.anchor_config = self.wireguard.getStatus()
        print(self.anchor_config)
        if(self.anchor_config != None):
           self.wireguard.start()
           time.sleep(2)

           for p in self.config['piers']:
              self.registerUrbit(p)

           self.save_config()
           return 0
        return 1


    def load_urbits(self):
        for p in self.config['piers']:
            data = None
            with open(f'settings/pier/{p}.json') as f:
                data = json.load(f)
            self._urbits[p] = UrbitDocker(data)
            self._minios[p] = MinIODocker(data)
        self.startMinIOs()

    def startMinIOs(self):
      for m in self._minios.values():
         m.start()
      self.minIO_on = True

    def stopMinIOs(self):
      for m in self._minios.values():
         m.stop()
      self.minIO_on = False

    def registerUrbit(self, patp):
       self.anchor_config = self.wireguard.getStatus()
       if(self.anchor_config != None):
          for ep in self.anchor_config['subdomains']:
             if(patp in ep['url']):
                 print(f"{patp} already exists")
                 return

       self.wireguard.registerService(f'{patp}','urbit')
       self.wireguard.registerService(f's3.{patp}','minio')
       self.anchor_config = self.wireguard.getStatus()

    def addUrbit(self, patp, urbit):
        self.config['piers'].append(patp)
        self.registerUrbit(patp)
        self.anchor_config = self.wireguard.getStatus()
        print(self.anchor_config)
        url = None
        http_port = None
        ames_port = None
        s3_port = None
        console_port = None


        for ep in self.anchor_config['subdomains']:
            if(f'{patp}.nativeplanet.live' == ep['url']):
                url = ep['url']
                http_port = ep['port']
            elif(f'ames.{patp}.nativeplanet.live' == ep['url']):
                ames_port = ep['port']
            elif(f's3.{patp}.nativeplanet.live' == ep['url']):
                s3_port = ep['port']
            elif(f'console.s3.{patp}.nativeplanet.live' == ep['url']):
                console_port = ep['port']

        urbit.setWireguardNetwork(url, http_port, ames_port, s3_port, console_port)

        self._urbits[patp] = urbit
        self._minios[patp] = MinIODocker(urbit.config)
        self.save_config()
        self.wireguard.start()
        self._minios[patp].start()
        urbit.start()
        

    def removeUrbit(self, patp):
        urb = self._urbits[patp]
        urb.removeUrbit()
        urb = self._urbits.pop(patp)
        
        time.sleep(2)
        minio = self._minios[patp]
        minio.removeMinIO()
        minio = self._minios.pop(patp)

        self.config['piers'].remove(patp)
        self.save_config()


    def getUrbits(self):
        urbits= []

        for urbit in self._urbits.values():
            u = dict()
            u['name'] = urbit.pier_name
            u['running'] = urbit.isRunning();
            u['s3_url'] = f"http://console.s3.{urbit.config['wg_url']}"
            if(urbit.config['network']=='wireguard'):
                u['url'] = f"http://{urbit.config['wg_url']}"
            else:
                u['url'] = f'http://nativeplanet.local:{urbit.config["http_port"]}'
            if(urbit.isRunning()):
                try:
                    u['code'] = urbit.get_code().decode('utf-8')
                except Exception as e:
                    print(e)
            else:
                u['code'] = ""

            u['network'] = urbit.config['network']
            
            urbits.append(u)

        return urbits
    
    def getContainers(self):
        minio = list(self._minios.keys())
        containers = list(self._urbits.keys())
        containers.append('wireguard')
        for m in minio:
            containers.append(f"minio_{m}")
        print(containers)
        return containers

    def switchUrbitNetwork(self, urbit_name):
        print(urbit_name)
        urbit = self._urbits[urbit_name]
        network = 'none'
        url = f"nativeplanet.local:{urbit.config['http_port']}"

        if(urbit.config['network'] == 'none'):
            network = 'wireguard'
            url = urbit.config['wg_url']

        urbit.setNetwork(network);
        time.sleep(2)

        

    def getOpenUrbitPort(self):
        http_port = 8080
        ames_port = 34343

        for u in self._urbits.values():
            if(u.config['http_port'] >= http_port):
                http_port = u.config['http_port']
            if(u.config['ames_port'] >= ames_port):
                ames_port = u.config['ames_port']

        return http_port+1, ames_port+1

    def getLogs(self, container):
        if container == 'wireguard':
            return self.wireguard.wg_docker.logs()
        if 'minio_' in container:
            return self._minios[container[6:]].logs()
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
        subprocess.run("rm privkey pubkey", shell =True)
   

    def save_config(self):
        with open(self.config_file, 'w') as f:
            json.dump(self.config, f, indent = 4)




if __name__ == '__main__':
    orchestrator = Orchestrator("settings/system.json")

