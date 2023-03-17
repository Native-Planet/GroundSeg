import docker
import json
import pathlib
import socket
import tarfile
import io

from log import Log

client = docker.from_env()

class WebUIDocker:
    def start(self, config, updater_info, arch):
        name = config['webui_name']
        tag = config['webui_version']
        v_tag = updater_info['tag']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:{v_tag}@sha256:{updater_info[sha]}"
        else:
            image = tag

        Log.log("WebUI: Attempting to start container")
        if self.get_container(name):
            self.remove_container(name)

        c = self.create_container(name, image, config)

        try:
            c.start()
            Log.log("WebUI: Successfully started container")
            return True
        except Exception as e:
            Log.log(f"WebUI: Failed to start container: {e}")
            return False


    def remove_container(self, name):
        Log.log("WebUI: Attempting to remove old container")
        c = self.get_container(name)
        if not c:
            Log.log("WebUI: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log("WebUI: Successfully removed container")
            return True

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log("WebUI: Container not found")
            return False


    def create_container(self, name, image, config):
        Log.log("WebUI: Attempting to create container")
        if self._pull_image(image):
                return self._build_container(name, image, config)


    def _pull_image(self, image):
        try:
            Log.log(f"WebUI: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"WebUI: Failed to pull {image}: {e}")
            return False


    def _build_container(self, name, image, config):
        try:
            port = config['port']
            hostname = socket.gethostname()

            Log.log("WebUI: Building container")

            c = client.containers.create(
                    image = image,
                    environment = [f"HOST_HOSTNAME={hostname}",f"PORT={port}"],
                    network='host',
                    name = name,
                    detach = True
                    )

            return c

        except Exception as e:
            Log.log(f"WebUI: Failed to build container: {e}")
            return False
