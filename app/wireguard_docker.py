import docker
import json

client = docker.from_env()

class WireguardDocker:
    _wireguard_img = "linuxserver/wireguard"
    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self,wireguard_config):
        self.config = wireguard_config
        client.images.pull(f'{self._wireguard_img}:{self.config["wireguard_version"]}')
        self.wireguard_name = self.config['wireguard_name']

        self.buildWireguard()
        self.running = (self.container.attrs['State']['Status'] == 'running' )



    def buildVolume(self):
        volumes = client.volumes.list()
        for v in volumes:
            if self.wireguard_name == v.name:
                self.volume = v
                return
        self.volume = client.volumes.create(name=self.wireguard_name)

    def buildContainer(self):
        containers = client.containers.list(all=True)

        for c in containers:
            if(self.wireguard_name == c.name):
                self.container = c
                return

        self.container = client.containers.create(
                image = f'{self._wireguard_img}:{self.config["wireguard_version"]}',
                name = self.wireguard_name,
                mounts = [self.mount],
                hostname = self.wireguard_name, 
                volumes = self.config['volumes'],
                cap_add = self.config['cap_add'],
                sysctls = self.config['sysctls'],
                detach=True)

    def buildWireguard(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/config', source =self.wireguard_name)
        self.buildContainer()
    
    def removeWireguard(self):
        wg.stop()
        self.container.remove()
        self.volume.remove()

    def addConfig(self, config):
        with open(f'{self._volume_directory}/{self.wireguard_name}/_data/wg0.conf', 'w') as f:
            f.write(config)

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
    filename = "settings/wireguard.json"
    f = open(filename)
    data = json.load(f)
    wg = WireguardDocker(data)
