import docker
#from log import Log

client = docker.from_env()

class WireguardDocker:

    def start(self, config, arch):
        name = config['wireguard_name']
        tag = config['wireguard_version']
        sha = f"{arch}_sha256"

        image = f"{config['repo']}:{tag}"
        if config[sha] != "":
            image = f"{image}@sha256:{config[sha]}"

        print("groundseg:wireguard:docker:start: Attempting to start container")
        c = self.get_container(name)
        if not c:
            c = self._create_container(name, image, config)
            if not c:
                return False

        if c.attrs['Config']['Image'] != image:
            print("groundseg:wireguard:docker:start: Container and config versions are mismatched")
            if self.remove_wireguard(name):
                c = self._create_container(name, image, config)
                if not c:
                    return False

        if c.status == "running":
            print("groundseg:wireguard:docker:start: Container already started")
            return True

        try:
            c.start()
            print("groundseg:wireguard:docker:start: Successfully started container")
            return True
        except:
            print("groundseg:wireguard:docker:start: Failed to start container")
            return False

    def stop(self, config):
        name = config['wireguard_name']
        print("groundseg:wireguard:docker:stop: Attempting to stop container")

        c = self.get_container(name)
        if not c:
            return False
        try:
            c.stop()
            print("groundseg:wireguard:docker:stop: Successfully stopped container")
            return True
        except:
            print("groundseg:wireguard:docker:stop: Failed to stop container")
            return False

    
    def remove_wireguard(self, name):
        if self.remove_container(name):
            return self.remove_volume(name)
        return False


    def add_config(self, vol_dir, config, wg0):
        print("groundseg:wireguard:docker:add_config Attempting to add wg0.conf")
        try:
            with open(f"{vol_dir}/{config['wireguard_name']}/_data/wg0.conf", "w") as f:
                f.write(wg0)
                f.close()
                return True
        except Exception as e:
            print(f"groundseg:wireguard:docker:add_config Failed to add wg0.conf: {e}")

        return False

    def logs(self, name):
        c = self.get_container(name)
        if not c:
            print("Wireguard: Failed to retrieve logs")
            return False
        return c.logs()


    def is_running(self, name):
        c = self.get_container(name)
        if not c:
            return False
        return c.status == "running"


    def remove_container(self, name):
        print("groundseg:wireguard:docker:remove_container: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            print("groundseg:wireguard:docker:remove_container: Container doesn't exist")
            return True
        else:
            c.remove(force=True)
            print("groundseg:wireguard:docker:remove_container: Successfully removed container")
            return True


    def remove_volume(self, name):
        print("groundseg:wireguard:docker:remove_volume: Attempting to remove volume")
        try:
            v = self._get_volume(name)
            v.remove()
            print("groundseg:wireguard:docker:remove_volume: Successfully removed volume")
        except:
            print("groundseg:wireguard:docker:remove_volume: Volume doesn't exist")
        return True

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            print("Wireguard: Container not found")
            return False

    def _create_container(self, name, image, config):
        print("Wireguard: Attempting to create container")
        if self._pull_image(image):
            v = self._build_volume(name)
            if v:
                print("Wireguard: Creating Mount object")
                mount = docker.types.Mount(target='/config', source=name)
                return self._build_container(name, image, mount, config)


    def _pull_image(self, image):
        try:
            print(f"Wireguard: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            print(f"Wireguard: Failed to pull {image}: {e}")
            return False


    def _get_volume(self, name):
        try:
            v = client.volumes.get(name)
            print("Wireguard: Volume found")
            return v
        except:
            print("Wireguard: Volume not found")
            return False


    def _build_volume(self, name):
        v = self._get_volume(name)
        if v:
            return v
        else:
            try:
                print("Wireguard: Attempting to create new volume")
                v = client.volumes.create(name=name)
                print("Wireguard: Volume created")
                return v
            except Exception:
                print("Wireguard: Failed to create volume: {e}")
                return False


    def _build_container(self, name, image, mount, config):
        try:
            vols = config['volumes']
            sysctls = config['sysctls']
            cap_add = config['cap_add']

            print("Wireguard: Building container")
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
            print(f"Wireguard: Failed to build container: {e}")
            return False

    def full_logs(self, name):
        c = self.get_container(name)
        if not c:
            return False
        return c.logs()

    def wg_show(self, name):
        c = self.get_container(name)
        if c:
            if c.status == "running":
                try:
                    res = c.exec_run("wg show")
                    return res
                except Exception as e:
                    print(f"{name}: Unable to exec {command}: {e}")
        return False
