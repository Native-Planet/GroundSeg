import docker, sys, json, time, shutil

client = docker.from_env()

class MinIODocker:
    _minio_img = "quay.io/minio/minio"

    def __init__(self,minio_config):
        self.config = minio_config
        client.images.pull(f'{self._minio_img}:{self.config["minio_version"]}')
        self.minio_name = f"minio_{self.config['pier_name']}"

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
                c.stop()
                c.remove()

        self.run()

        return 0

    def buildMinIO(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/data', source =self.minio_name)
        self.buildContainer()
    
    def start(self):
        self.container.stop()
        self.container.remove()
        self.run()

    def run(self):

        console_port = self.config['wg_console_port']
        s3_port = self.config['wg_s3_port']
        command = f'server /data --console-address ":{console_port}" --address ":{s3_port}"'

        environment = [f"MINIO_ROOT_USER={self.config['pier_name']}", 
                      f"MINIO_ROOT_PASSWORD={self.config['minio_password']}",
                      f"MINIO_DOMAIN=s3.{self.config['wg_url']}",
                      f"MINIO_SERVER_URL=https://s3.{self.config['wg_url']}"]

        self.container = client.containers.run(
                image= f'{self._minio_img}:{self.config["minio_version"]}',
                command=command, 
                name = self.minio_name,
                environment = environment,
                network = f'container:wireguard',
                mounts = [self.mount],
                detach=True)

        self.container.exec_run('mkdir -p /data/bucket')

    def stop(self):
        self.container.stop()

    def logs(self):
        return self.container.logs()

    def remove_minio(self):
        self.stop()
        self.container.remove()
        self.volume.remove()
