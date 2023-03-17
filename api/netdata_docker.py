import docker
from log import Log

client = docker.from_env()
class NetdataDocker:
    def start(self, config, updater_info, arch):
        name = config['netdata_name']
        tag = config['netdata_version']
        v_tag = updater_info['tag']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:{v_tag}@sha256:{updater_info[sha]}"
        else:
            image = tag

        Log.log("Netdata: Attempting to start container")
        c = self.get_container(name)
        if not c:
            c = self.create_container(name, image, config)
            if not c:
                return False

        if c.attrs['Config']['Image'] != image:
            Log.log("Netdata: Container and config versions are mismatched")
            if self.remove_container(name):
                c = self.create_container(name, image, config)
                if not c:
                    return False

        if c.status == "running":
            Log.log("Netdata: Container already started")
            return True

        try:
            c.start()
            Log.log("Netdata: Successfully started container")
            return True
        except:
            Log.log("Netdata: Failed to start container")
            return False

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log("Netdata: Container not found")
            return False

    def create_container(self, name, image, config):
        Log.log("Netdata: Attempting to create container")
        if self.pull_image(image):
            return self.build_container(name, image, config)

    def remove_container(self, name):
        Log.log("Netdata: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            Log.log("Netdata: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log("Netdata: Successfully removed container")
            return True

    def pull_image(self, image):
        try:
            Log.log(f"Netdata: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"Netdata: Failed to pull {image}: {e}")
            return False

    def build_container(self, name, image, config):
        try:
            vols = config['volumes']
            cap_add = config['cap_add']
            restart = config['restart']

            Log.log("Netdata: Building container")
            c = client.containers.create(
                    image = image,
                    name = name,
                    hostname = name, 
                    volumes = vols,
                    cap_add = cap_add,
                    ports = {'19999':str(config['port'])},
                    restart_policy = {"always": config['restart']},
                    security_opt = [config['security_opt']],
                    detach=True)
            return c

        except Exception as e:
            Log.log(f"Netdata: Failed to build container: {e}")
            return False

    def full_logs(self, name):
        c = self.get_container(name)
        if not c:
            return False
        return c.logs()
