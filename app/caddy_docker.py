import docker
import json
import pathlib

client = docker.from_env()

class CaddyDocker:
    _caddy_img = "caddy:latest"

    def __init__(self):
        client.images.pull(f'{self._caddy_img}')
        self.caddy_name = "ui_caddy"

        self.buildCaddy()


    def buildContainer(self):
        containers = client.containers.list(all=True)

        for c in containers:
            if(self.caddy_name == c.name):
                c.stop()
                c.remove()

        
        self.container = client.containers.create(
                    image= f'{self._caddy_img}',
                    ports = {'80/tcp':80}, 
                    name = self.caddy_name,
                    volumes = [f'{pathlib.Path(__file__).parent.resolve()}/ui/:/usr/share/caddy/'],
                    detach=True)

    def buildCaddy(self):
        self.buildContainer()
    
    def start(self):
        self.container.start()

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def removeCaddy(self):
        self.stop()
        self.container.remove()



     



if __name__ == '__main__':
    caddy = CaddyDocker()
