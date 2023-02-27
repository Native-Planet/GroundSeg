import docker
from log import Log

client = docker.from_env()

class MinIODocker:
    def start(self, name, updater_info, config, arch):
        tag = config['minio_version']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"
        else:
            image = f"{updater_info['repo']}:{updater_info['tag']}"

        Log.log(f"{name}: Attempting to start container")
        # Get container
        c = self.get_container(name)
        if not c:
            c = self.create_container(name, image, config)
            if not c:
                return False

        # Check for version match
        if c.attrs['Config']['Image'] != image:
            Log.log(f"{name}: Container and config versions are mismatched")
            if self.remove_container(name):
                c = self.create_container(name, image, config)
                if not c:
                    return False

        # Get status
        if c.status == "running":
            Log.log(f"{name}: Container already started")
            return True

        try:
            c.start()
            Log.log(f"{name}: Successfully started container")
            return self.exec(name, 'mkdir -p /data/bucket')
        except:
            Log.log(f"{name}: Failed to start container")
            return False

    def stop(self, name):
        Log.log(f"{name}: Attempting to stop container")
        c = self.get_container(name)
        if c:
            try:
                c.stop()
            except Exception as e:
                Log.log(f"{name}: Failed to stop container")
                return False

        Log.log(f"{name}: Container stopped")
        return True

    def delete(self, name):
        if self.remove_container(name):
            return self.delete_volume(name)

    def delete_volume(self, name):
        Log.log(f"{name}: Attempting to delete volume")
        v = self.get_volume(name)
        if not v:
            return True
        try:
            v.remove(force=True)
            Log.log(f"{name}: Volume deleted")
            return True
        except Exception as e:
            Log.log(f"{name}: Error removing volume: {e}")

        return False

    def exec(self, name, command):
        c = self.get_container(name)
        if c:
            try:
                Log.log(f"{name}: Sending command: {command}")
                x = c.exec_run(command)
                Log.log(f"{name}: Result: {x}")
                return True
            except Exception as e:
                Log.log(f"{name}: Unable to exec {command}: {e}")

        return False

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log(f"{name}: Container not found")
            return False

    def create_container(self, name, image, config):
        Log.log(f"{name}: Attempting to create container")
        if self.pull_image(name, image):
            v = self.build_volume(name)
            if v:
                Log.log(f"{name}: Creating Mount object")
                mount = docker.types.Mount(target='/data', source=name)
                return self.build_container(name, image, mount, config)

    def remove_container(self, name):
        Log.log(f"{name}: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            Log.log(f"{name}: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log(f"{name}: Successfully removed container")
            return True

    def pull_image(self, name, image):
        try:
            Log.log(f"{name}: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"{name}: Failed to pull {image}: {e}")
            return False

    def get_volume(self, name):
        try:
            v = client.volumes.get(name)
            Log.log(f"{name}: Volume found")
            return v
        except:
            Log.log(f"{name}: Volume not found")
            return False


    def build_volume(self, name):
        v = self.get_volume(name)
        if v:
            return v
        else:
            try:
                Log.log(f"{name}: Attempting to create new volume")
                v = client.volumes.create(name=name)
                Log.log(f"{name}: Volume created")
                return v
            except Exception as e:
                Log.log(f"{name}: Failed to create volume: {e}")
                return False

    def build_container(self, name, image, mount, config):
        try:
            console_port = config['wg_console_port']
            s3_port = config['wg_s3_port']
            command = f'server /data --console-address ":{console_port}" --address ":{s3_port}"'

            environment = [f"MINIO_ROOT_USER={config['pier_name']}", 
                          f"MINIO_ROOT_PASSWORD={config['minio_password']}",
                          f"MINIO_DOMAIN=s3.{config['wg_url']}",
                          f"MINIO_SERVER_URL=https://s3.{config['wg_url']}"]

            Log.log(f"{name}: Building container")
            c = client.containers.create(
                    image = image,
                    command = command, 
                    name = name,
                    environment = environment,
                    network = 'container:wireguard',
                    mounts = [mount],
                    detach=True)
            return c

        except Exception as e:
            Log.log(f"{name}: Failed to build container: {e}")
            return False
