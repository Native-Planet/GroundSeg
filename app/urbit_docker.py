import docker
import json
import shutil

client = docker.from_env()
default_pier_config = json.load(open('settings/default_urbit.json'))

class UrbitDocker:
    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self,pier_config):
        self.config = pier_config


        client.images.pull(f'tloncorp/urbit:{self.config["urbit_version"]}')
        self.pier_name = self.config['pier_name']
        self.buildUrbit()
        self.running = (self.container.attrs['State']['Status'] == 'running' )

    def buildVolume(self):
        volumes = client.volumes.list()
        for v in volumes:
            if self.pier_name == v.name:
                self.volume = v
                shutil.copy('./settings/start_urbit.sh', 
                        f'{self._volume_directory}/{self.pier_name}/_data/start_urbit.sh')
                return
        self.volume = client.volumes.create(name=self.pier_name)
        shutil.copy('./settings/start_urbit.sh', 
                f'{self._volume_directory}/{self.pier_name}/_data/start_urbit.sh')

    def buildContainer(self):
        containers = client.containers.list(all=True)
        for c in containers:
            if(self.pier_name == c.name):
                self.container = c
                return
        if(self.config["network"] != "none"):
            command = f'/urbit/start_urbit.sh --http-port={self.config["wg_http_port"]} \
                                          --port={self.config["wg_ames_port"]}'
            self.container = client.containers.create(
                                    f'tloncorp/urbit:{self.config["urbit_version"]}',
                                    command=command, 
                                    name = self.pier_name,
                                    network = f'container:{self.config["network"]}',
                                    mounts = [self.mount],
                                    detach=True)
        else:
            self.container = client.containers.create(
                                    f'tloncorp/urbit:{self.config["urbit_version"]}',
                                    ports = {'80/tcp':self.config['http_port'], 
                                             '34343/udp':self.config['ames_port']},
                                    name = self.pier_name,
                                    mounts = [self.mount],
                                    detach=True)


    def setWireguardNetwork(self, url, http_port, ames_port, s3_port):
        self.config['wg_url'] = url
        self.config['wg_http_port'] = http_port
        self.config['wg_ames_port'] = ames_port
        self.config['wg_s3_port'] = s3_port
        #self.config['network'] = 'wireguard'
        self.save_config()

        running = False
        
        if(self.isRunning()):
            self.stop()
            running = True

        self.container.remove()
        
        self.buildContainer()
        if(running):
            self.start()

    def setNetwork(self, network):
        if(self.config['network'] == network):
            return

        running = False
        if(self.isRunning()):
            self.stop()
            running = True
        
        self.container.remove()
        self.config['network']=network
        self.save_config()

        self.buildContainer()

        if(running):
            self.start()

    def save_config(self):
        with open(f'settings/{self.pier_name}.json', 'w') as f:
            json.dump(self.config, f, indent = 4)



    def buildUrbit(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/urbit/', source =self.pier_name)
        self.buildContainer()
    
    def removeUrbit(self):
        self.stop()
        self.container.remove()
        self.volume.remove()

    def addKey(self, key_value):
        with open(f'{self._volume_directory}/{self.pier_name}/_data/{self.pier_name}.key', 'w') as f:
            f.write(key_value)

    def copyFolder(self,folder_loc):
        from distutils.dir_util import copy_tree
        copy_tree(folder_loc,f'{self._volume_directory}/{self.pier_name}/_data/')

    def get_code(self):
        return self.container.exec_run('/bin/get-urbit-code').output.strip()

    def reset_code(self):
        return self.container.exec_run('/bin/reset-urbit-code').output.strip()

    def start(self):
        self.container.start()
        self.running=True

    def stop(self):
        self.container.stop()
        self.running=False

    def logs(self):
        return self.container.logs()
    
    def isRunning(self):
        return self.running




if __name__ == '__main__':
    filename = "settings/walzod-fogsed-mopfel-winrux.json"
    f = open(filename)
    data = json.load(f)
    urdock = UrbitDocker(data)
    urdock.start()
    import time
    time.sleep(2)
    print(urdock.logs().decode('utf-8'))
    urdock.stop()
