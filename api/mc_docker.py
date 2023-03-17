import docker
from log import Log

client = docker.from_env()

class MCDocker:
    def start(self, name, updater_info, arch):
        sha = f"{arch}_sha256"
        v_tag = updater_info['tag']
        image = f"{updater_info['repo']}:{v_tag}@sha256:{updater_info[sha]}"

        Log.log("MC: Attempting to start container")
        c = self.get_container(name)
        if c:
            self.remove_container(name)

        c = self.create_container(name, image)
        if not c:
            return False

        if c.attrs['Config']['Image'] != image:
            Log.log("MC: Container and config versions are mismatched")
            if self.remove_container(name):
                c = self.create_container(name, image)
                if not c:
                    return False

        if c.status == "running":
            Log.log("MC: Container already started")
            return True

        try:
            c.start()
            Log.log("MC: Successfully started container")
            return True
        except:
            Log.log("MC: Failed to start container")
            return False

    def stop(self, name):
        Log.log("MC: Attempting to stop container")
        c = self.get_container(name)
        if not c:
            return False
        c.stop()
        Log.log("MC: Container stopped")
        return True

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log("MC: Container not found")
            return False

    def create_container(self, name, image):
        Log.log("MC: Attempting to create container")
        if self.pull_image(image):
            return self.build_container(name, image)

    def pull_image(self, image):
        try:
            Log.log(f"MC: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"MC: Failed to pull {image}: {e}")
            return False

    def build_container(self, name, image):
        try:
            Log.log("MC: Building container")
            c = client.containers.create(
                    image = image,
                    network = 'container:wireguard',
                    entrypoint = '/bin/bash',
                    stdin_open = True,
                    name = name,
                    detach=True)
            return c

        except Exception as e:
            Log.log(f"MC: Failed to build container: {e}")
            return False

    def remove_container(self, name):
        Log.log("MC: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            Log.log("MC: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log("MC: Successfully removed container")
            return True

    def exec(self, name, command):
        Log.log(f"{name}: Executing command")
        c = self.get_container(name)
        if c:
            try:
                x = c.exec_run(command)
                Log.log(f"{name}: Output: {x.output.decode('utf-8').strip()}")

                return True
            except Exception as e:
                Log.log(f"{name}: Unable to exec command: {e}")

        return False
