import docker
import json
import pathlib

client = docker.from_env()

class WatchtowerDocker:
    _wt_img = "containrrr/watchtower:latest"

    def __init__(self, mode):
        client.images.pull(f'{self._wt_img}')

        self.buildContainer(mode)

    def buildContainer(self, mode):
        containers = client.containers.list(all=True)

        env_args = {
               'WATCHTOWER_POLL_INTERVAL': 90,
               'WATCHTOWER_LABEL_ENABLE': True,
               'WATCHTOWER_CLEANUP': True
               }
        if mode == 'manual':
           env_args['WATCHTOWER_MONITOR_ONLY'] = True

        for c in containers:
            if(c.name == 'watchtower'):
                c.stop()
                c.remove()
        
        self.container = client.containers.create(
                    name = 'watchtower',
                    image = f'{self._wt_img}',
                    volumes = [f'/var/run/docker.sock:/var/run/docker.sock'],
                    environment = env_args,
                    detach = True)

        if mode == 'off':
            self.stop()
        if mode == 'auto':
            self.start()

    def start(self):
        self.container.start()

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def remove(self):
        self.stop()
        self.container.remove()
