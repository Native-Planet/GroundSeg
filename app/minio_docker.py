import docker
import json

client = docker.from_env()

class MinIODocker:
    _minio_img = "quay.io/minio/minio"

    def __init__(self,minio_config):
        self.config = minio_config
        client.images.pull(f'{self._minio_img}:{self.config["minio_version"]}')
        self.minio_name = self.config['minio_name']

        self.buildMinIO()



    def buildVolume(self):
        volumes = client.volumes.list()
        for v in volumes:
            if self.minio_name == v.name:
                self.volume = v
                return
        self.volume = client.volumes.create(name=self.minio_name)

    def buildContainer(self):
        containers = client.containers.list(all=True)

        for c in containers:
            if(self.minio_name == c.name):
                self.container = c
                return

        self.container = client.containers.create(
                image = f'{self._minio_img}:{self.config["minio_version"]}',
                command = 'server /data --console-address ":9001"',
                ports = {'9000/tcp':self.config['http_port'], 
                         '9001/tcp':self.config['console_port']},
                name = self.minio_name,
                mounts = [self.mount],
                network_mode = f'container:{self.config["network"]}',
                detach=True)

    def buildMinIO(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/data', source =self.minio_name)
        self.buildContainer()
    
    def start(self):
        self.container.start()

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def removeMinIO(self):
        self.volume.remove()
        self.container.remove()



     



if __name__ == '__main__':
    filename = "minio.json"
    f = open(filename)
    data = json.load(f)
    minio = MinIODocker(data)
