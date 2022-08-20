import docker
import json

client = docker.from_env()

class UrbitDocker:
    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self,pier_config):
        self.config = pier_config


        client.images.pull(f'tloncorp/urbit:{self.config["urbit_version"]}')
        self.pier_name = self.config['pier_name']
        self.buildUrbit()

    def buildVolume(self):
        volumes = client.volumes.list()
        for v in volumes:
            if self.pier_name == v.name:
                self.volume = v
                return
        self.volume = client.volumes.create(name=self.pier_name)

    def buildContainer(self):
        containers = client.containers.list(all=True)
        for c in containers:
            if(self.pier_name == c.name):
                self.container = c
                return
        self.container = client.containers.create(f'tloncorp/urbit:{self.config["urbit_version"]}',
                                    ports = {'80/tcp':self.config['http_port'], 
                                             '34343/udp':self.config['ames_port']},
                                    name = self.pier_name,
                                    mounts = [self.mount],
                                    detach=True)


    def buildUrbit(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/urbit/', source =self.pier_name)
        self.buildContainer()
    
    def removeUrbit(self):
        self.volume.remove()
        self.container.remove()

    def addKey(self, key_value):
        with open(f'{self._volume_directory}/{self.pier_name}/_data/{self.pier_name}.key', 'w') as f:
            f.write(key_value)

    def start(self):
        self.container.start()

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def get_code(self):
        return self.container.exec_run('/bin/get-urbit-code').output.strip()

    def reset_code(self):
        return self.container.exec_run('/bin/reset-urbit-code').output.strip()



if __name__ == '__main__':
    filename = "famwyl-lavlyr-mopfel-winrux.json"
    f = open(filename)
    data = json.load(f)
    urdock = UrbitDocker(data)
