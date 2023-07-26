import docker
#from log import Log

client = docker.from_env()

class MCDocker:
    def start(self, config, arch):
        name = config['mc_name']
        tag = config['mc_version']
        sha = f"{arch}_sha256"

        image = f"{config['repo']}:{tag}"
        if config[sha] != "":
            image = f"{image}@sha256:{config[sha]}"

        print("MC: Attempting to start container")
        c = self.get_container(name)
        if c:
            self.remove_container(name)

        c = self.create_container(name, image)
        if not c:
            return False

        if c.status == "running":
            print("MC: Container already started")
            return True

        try:
            c.start()
            print("MC: Successfully started container")
            return True
        except:
            print("MC: Failed to start container")
            return False

    def stop(self, name):
        print("MC: Attempting to stop container")
        c = self.get_container(name)
        if not c:
            return False
        c.stop()
        print("MC: Container stopped")
        return True

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            print("MC: Container not found")
            return False

    def create_container(self, name, image):
        print("MC: Attempting to create container")
        if self.pull_image(image):
            return self.build_container(name, image)

    def pull_image(self, image):
        try:
            print(f"MC: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            print(f"MC: Failed to pull {image}: {e}")
            return False

    def build_container(self, name, image):
        try:
            print("MC: Building container")
            c = client.containers.create(
                    image = image,
                    network = 'container:wireguard',
                    entrypoint = '/bin/bash',
                    stdin_open = True,
                    name = name,
                    detach=True)
            return c

        except Exception as e:
            print(f"MC: Failed to build container: {e}")
            return False

    def remove_container(self, name):
        print("MC: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            print("MC: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            print("MC: Successfully removed container")
            return True

    def exec(self, name, command):
        print("MC: Executing command")
        c = self.get_container(name)
        if c:
            try:
                x = c.exec_run(command)
                print(f"MC: Output: {x.output.decode('utf-8').strip()}")

                return True
            except Exception as e:
                print(f"MC: Unable to exec command: {e}")

        return False
