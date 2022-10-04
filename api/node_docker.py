import docker
import json
import pathlib

client = docker.from_env()

class NodeDocker:
    _node_img = "node:latest"

    def __init__(self):
        client.images.pull(f'{self._node_img}')
        self.node_name = "ui_nodejs"

        self.buildNode()


    def buildContainer(self):
        containers = client.containers.list(all=True)

        for c in containers:
            if(self.node_name == c.name):
                c.stop()
                c.remove()

        
        self.container = client.containers.create(
                    command='node /home/node/app/build/index.js',
                    image= f'{self._node_img}',
                    environment = ["PORT=80"],
                    network='host',
                    name = self.node_name,
                    volumes = [f'{pathlib.Path(__file__).parent.resolve()}/ui/:/home/node/app'],
                    detach=True)


    def buildNode(self):
        self.buildContainer()
    
    def start(self):
        self.container.start()

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def removeNode(self):
        self.stop()
        self.container.remove()



     



if __name__ == '__main__':
    node = NodeDocker()
