import docker
import json

from log import Log

client = docker.from_env()

class WireguardDocker:

    def start(self, config, updater_info, arch):
        name = config['wireguard_name']
        tag = config['tag']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"
        else:
            image = f"{updater_info['repo']}:{updater_info['tag']}"

        Log.log("Wireguard: Attempting to start container")
        c = self._get_container(name)
        if not c:
            c = self._create_container(name, image, config)
            if not c:
                return False

        if c.attrs['Config']['Image'] != image:
            Log.log("Wireguard: Container and config versions are mismatched")
            if self.remove_wireguard(name):
                c = self._create_container(name, image, config)
                if not c:
                    return False

        if c.status == "running":
            Log.log("Wireguard: Container already started")
            return True

        try:
            c.start()
            Log.log("Wireguard: Successfully started container")
            return True
        except:
            Log.log("Wireguard: Failed to start container")
            return False


    def stop(self, config):
        name = config['wireguard_name']
        Log.log("Wireguard: Attempting to stop container")

        c = self._get_container(name)
        if not c:
            return False
        try:
            c.stop()
            Log.log("Wireguard: Successfully stopped container")
            return True
        except:
            Log.log("Wireguard: Failed to stop container")
            return False

    
    def remove_wireguard(self, name):
        if self._remove_container(name):
            return self._remove_volume(name)
        return False


    def add_config(self, config, wg0):
        Log.log("Wireguard: Attempting to add wg0.conf")
        try:
            with open(f"{config['volume_dir']}/{config['wireguard_name']}/_data/wg0.conf", "w") as f:
                f.write(wg0)
                f.close()
                return True
        except Exception as e:
            Log.log(f"Wireguard: Failed to add wg0.conf: {e}")

        return False

    def logs(self, name):
        c = self._get_container(name)
        if not c:
            Log.log("Wireguard: Failed to retrieve logs")
            return False
        return c.logs()


    def is_running(self, name):
        c = self._get_container(name)
        if not c:
            return False
        return c.status == "running"


    def _remove_container(self, name):
        Log.log("Wireguard: Attempting to remove container")
        c = self._get_container(name)
        if not c:
            Log.log("Wireguard: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log("Wireguard: Successfully removed container")
            return True


    def _remove_volume(self, name):
        Log.log("Wireguard: Attempting to remove volume")
        try:
            v = self._get_volume(name)
            v.remove()
            Log.log("Wireguard: Successfully removed volume")
            return True
        except:
            Log.log("Wireguard: Failed to remove volume")
            return False


    def _get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log("Wireguard: Container not found")
            return False


    def _create_container(self, name, image, config):
        Log.log("Wireguard: Attempting to create container")
        if self._pull_image(image):
            v = self._build_volume(name)
            if v:
                Log.log("Wireguard: Creating Mount object")
                mount = docker.types.Mount(target='/config', source=name)
                return self._build_container(name, image, mount, config)


    def _pull_image(self, image):
        try:
            Log.log(f"Wireguard: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"Wireguard: Failed to pull {image}: {e}")
            return False


    def _get_volume(self, name):
        try:
            v = client.volumes.get(name)
            Log.log("Wireguard: Volume found")
            return v
        except:
            Log.log("Wireguard: Volume not found")
            return False


    def _build_volume(self, name):
        v = self._get_volume(name)
        if v:
            return v
        else:
            try:
                Log.log("Wireguard: Attempting to create new volume")
                v = client.volumes.create(name=name)
                Log.log("Wireguard: Volume created")
                return v
            except Exception as e:
                Log.log("Wireguard: Failed to create volume: {e}")
                return False


    def _build_container(self, name, image, mount, config):
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
