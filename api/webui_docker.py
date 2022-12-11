import docker
import json
import pathlib
import socket

client = docker.from_env()

class WebUIDocker:
    _webui_img = "nativeplanet/groundseg-webui:edge"

    def __init__(self,port):
        client.images.pull(self._webui_img)

        containers = client.containers.list(all=True)

        for c in containers:
            if c.name == 'groundseg-webui':
                try:
                    c.stop()
                    c.remove()
                except Exception as e:
                    print(f"Webui removal error: {e}", file=sys.stderr)

        self.container = client.containers.run(
                    image= f'{self._webui_img}',
                    environment = [f"HOST_HOSTNAME={socket.gethostname()}",f"PORT={port}"],
                    labels = {"com.centurylinklabs.watchtower.enable":"true"},
                    network='host',
                    name = 'groundseg-webui',
                    detach=True
                    )
