import docker
import json
import pathlib
import socket

client = docker.from_env()

class WebUIDocker:
    def __init__(self, port, update_branch):

        webui_img = f"nativeplanet/groundseg-webui:{update_branch}"

        client.images.pull(webui_img)

        containers = client.containers.list(all=True)

        for c in containers:
            if c.name == 'groundseg-webui':
                try:
                    c.stop()
                    c.remove()
                except Exception as e:
                    print(f"Webui removal error: {e}", file=sys.stderr)

        self.container = client.containers.run(
                    image= webui_img,
                    environment = [f"HOST_HOSTNAME={socket.gethostname()}",f"PORT={port}"],
                    network='host',
                    name = 'groundseg-webui',
                    detach=True
                    )
