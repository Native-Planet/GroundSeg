import docker
client = docker.from_env()

class NetdataDocker:
    def start(self, config, arch):
        name = config['netdata_name']
        tag = config['netdata_version']
        sha = f"{arch}_sha256"

        image = f"{config['repo']}:{tag}"
        if config[sha] != "":
            image = f"{image}@sha256:{config[sha]}"

        print("groundseg:netdata:docker:start: Attempting to start container")
        c = self.get_container(name)
        if not c:
            c = self.create_container(name, image, config)
            if not c:
                return False

        if c.attrs['Config']['Image'] != image:
            print("groundseg:netdata:docker:start: Container and config versions are mismatched")
            if self.remove_container(name):
                c = self.create_container(name, image, config)
                if not c:
                    return False

        if c.status == "running":
            print("groundseg:netdata:docker:start: Container already started")
            return True

        try:
            c.start()
            print("groundseg:netdata:docker:start: Successfully started container")
            return True
        except:
            print("groundseg:netdata:docker:start: Failed to start container")
        return False

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            print("groundseg:netdata:docker:get_container: Container not found")
        return False

    def create_container(self, name, image, config):
        print("groundseg:netdata:docker:create_container Attempting to create container")
        if self.pull_image(image):
            return self.build_container(name, image, config)

    def remove_container(self, name):
        print("groundseg:netdata:docker:remove_container: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            print("groundseg:netdata:docker:remove_container: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            print("groundseg:netdata:docker:remove_container: Successfully removed container")
        return True

    def pull_image(self, image):
        try:
            print(f"groundseg:netdata:docker:pull_image: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            print(f"groundseg:netdata:docker:pull_image: Failed to pull {image}: {e}")
        return False

    def build_container(self, name, image, config):
        try:
            vols = config['volumes']
            cap_add = config['cap_add']

            print("groundseg:netdata:docker:build_container: Building container")
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
            print(f"groundseg:netdata:docker:build_container: Failed to build container: {e}")
        return False

    def full_logs(self, name):
        c = self.get_container(name)
        if not c:
            return False
        return c.logs()
