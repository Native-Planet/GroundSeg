import docker
import json

from log import Log

client = docker.from_env()

class WireguardDocker:

    def start(self, config, updater_info):
        name = config['wireguard_name']
        image = f"{updater_info['repo']}:{updater_info['tag']}"

        Log.log("Wireguard: Attempting to start container")
        c = self.get_container(name)
        if not c:
            c = self.create_container(name, image, config)
            if not c:
                return False
        try:
            c.start()
            return True
        except:
            return False

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            Log.log("Wireguard: Container found")
            return c
        except:
            Log.log("Wireguard: Container not found")
            return False

    def create_container(self, name, image, config):
        Log.log("Wireguard: Attempting to create container")
        if self.pull_image(image):
            v = self.build_volume(name)
            if v:
                mount = docker.types.Mount(target='/config', source=name)
                return self.build_container(name, image, mount, config)

    def pull_image(self, image):
        try:
            client.images.pull(image)
            Log.log(f"Wireguard: Pulling {image}")
            return True
        except Exception as e:
            Log.log(f"Wireguard: Failed to pull {image}: {e}")
            return False

    def build_volume(self, name):
        try:
            v = client.volumes.get(name)
            Log.log("Wireguard: Volume found")
            return v
        except:
            Log.log("Wireguard: Volume not found")
            try:
                Log.log("Wireguard: Attempting to create new volume")
                v = client.volumes.create(name=name)
                Log.log("Wireguard: Volume created")
                return v
            except Exception as e:
                Log.log("Wireguard: Failed to create volume: {e}")
                return False

    def build_container(self, name, image, mount, config):
        try:
            vols = config['volumes']
            sysctls = config['sysctls']
            cap_add = config['cap_add']

            Log.log("Wireguard: Building container")
            c = client.containers.create(
                    image = image,
                    name = name,
                    mounts = [mount],
                    hostname = name, 
                    volumes = vols,
                    cap_add = cap_add,
                    sysctls = sysctls,
                    detach=True)
            return c
        except Exception as e:
            Log.log(f"Wireguard: Failed to build container: {e}")
            return False

    '''
    def buildWireguard(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/config', source =self.wireguard_name)
        self.buildContainer()
    
    def removeWireguard(self):
        wg.stop()
        self.container.remove()
        self.volume.remove()

    def add_config(self, config):
        with open(f'{self._volume_directory}/{self.wireguard_name}/_data/wg0.conf', 'w') as f:
            f.write(config)

    def start(self):
        self.container.start()
        self.running=True
        return 0

    def stop(self):
        self.container.stop()
        self.running=False

    def logs(self):
        return self.container.logs()

    def is_running(self):
        return self.running
    '''
