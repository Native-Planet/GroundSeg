# Modules
import docker

# GroundSeg modules
from utils import Utils
from log import Log

client = docker.from_env()

class UrbitDocker:

    def start(self, config, updater_info, arch, vol_dir):
        patp = config['pier_name']
        Log.log(f"{patp}: Attempting to start container")
        tag = config['urbit_version']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"
        else:
            image = f"{updater_info['repo']}:{updater_info['tag']}"

        # Check if patp is valid
        if not Utils.check_patp(patp):
            Log.log(f"{patp}: Invalid patp")
            return "invalid"

        # Get container
        c = self.get_container(patp)
        if not c:
            if self.create(config, updater_info, arch, vol_dir, key=''):
                c = self.get_container(patp)
                if not c:
                    return "failed"

        if c.attrs['Config']['Image'] != image:
            Log.log(f"{patp}: Container and config versions are mismatched")
            if self.remove_container(patp):
                c = self.create(config, updater_info, arch, vol_dir, '')
                if not c:
                    return "failed"

        # Get status
        if c.status == "running":
            Log.log(f"{patp}: Container already started")
            return "succeeded"

        # Start ship container
        try:
            with open(f'{vol_dir}/{patp}/_data/start_urbit.sh', 'w') as f:
                script = Utils.start_script()
                f.write(script)
                f.close()
            c.start()
            Log.log(f"{patp}: Successfully started container")
            return "succeeded"
        except Exception as e:
            Log.log(f"{patp}: Failed to start container: {e}")
            return "failed"

    def stop(self, patp):
        Log.log(f"{patp}: Attempting to stop container")
        c = self.get_container(patp)
        if c:
            try:
                c.stop()
            except Exception as e:
                Log.log(f"{patp}: Failed to stop container")
                return False

        Log.log(f"{patp}: Container stopped")
        return True

    def get_container(self, patp):
        try:
            c = client.containers.get(patp)
            return c
        except:
            Log.log(f"{patp}: Container not found")
            return False


    def create(self, config, updater_info, arch, vol_dir, key=''):
        patp = config['pier_name']
        tag = config['urbit_version']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"
        else:
            image = f"{updater_info['repo']}:{updater_info['tag']}"

        Log.log(f"{patp}: Attempting to create container")

        if self._pull_image(image, patp):
            v = self._build_volume(patp, vol_dir)
            if v:
                Log.log(f"{patp}: Creating Mount object")
                mount = docker.types.Mount(target = '/urbit/', source=patp)
                if self._build_container(patp, image, mount, config):
                    return self.add_key(key, patp, vol_dir)

    def delete(self, patp):
        if self.remove_container(patp):
            return self.delete_volume(patp)

    def remove_container(self, patp):
        Log.log(f"{patp}: Attempting to delete container")
        c = self.get_container(patp)
        if not c:
            return True
        try:
            c.remove(force=True)
            Log.log(f"{patp}: Container deleted")
            return True
        except Exception as e:
            Log.log(f"{patp}: Failed to delete container: {e}")

        return False

    def delete_volume(self, patp):
        Log.log(f"{patp}: Attempting to delete volume")
        v = self._get_volume(patp)
        if not v:
            return True
        try:
            v.remove(force=True)
            Log.log(f"{patp}: Volume deleted")
            return True
        except Exception as e:
            Log.log(f"{patp}: Error removing volume: {e}")

        return False

    def add_key(self, key, patp, vol_dir):
        if len(key) > 0:
            Log.log(f"{patp}: Attempting to add key")
            try:
                with open(f'{vol_dir}/{patp}/_data/{patp}.key', 'w') as f:
                    f.write(key)
                    f.close()
                return True
            except Exception as e:
                Log.log(f"{patp}: Failed to add key: {e}")

            return False
        return True

    def full_logs(self, patp):
        c = self.get_container(patp)
        if not c:
            return False
        return c.logs()

    def exec(self, patp, command):
        c = self.get_container(patp)
        if c:
            try:
                res = c.exec_run(command)
                return res
            except Exception as e:
                Log.log(f"{patp}: Unable to exec {command}: {e}")

        return False

    def _pull_image(self, image, patp):
        try:
            Log.log(f"{patp}: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"{patp}: Failed to pull {image}: {e}")
            return False

    def _get_volume(self, patp):
        try:
            v = client.volumes.get(patp)
            Log.log(f"{patp}: Volume found")
            return v
        except:
            Log.log(f"{patp}: Volume not found")
            return False


    def _build_volume(self, patp, vol_dir):
        v = self._get_volume(patp)
        if v:
            return v
        else:
            try:
                Log.log(f"{patp}: Attempting to create new volume")
                v = client.volumes.create(name=patp)
                Log.log(f"{patp}: Volume created")
                return v

            except Exception as e:
                Log.log(f"{patp}: Failed to create volume: {e}")
                return False


    def _build_container(self, patp, image, mount, config):
        try:
            Log.log(f"{patp}: Building container")
            command = f'bash /urbit/start_urbit.sh --loom={config["loom_size"]}'

            if config["network"] != "none":
                Log.log(f"{patp}: Network is set to wireguard")
                http = f"--http-port={config['wg_http_port']}"
                ames = f"--port={config['wg_ames_port']}"
                command = f"{command} {http} {ames}"

                c = client.containers.create(
                        image = image,
                        command = command, 
                        name = patp,
                        network = f'container:{config["network"]}',
                        mounts = [mount],
                        detach=True)
            else:
                c = client.containers.create(
                        image = image,
                        command = command, 
                        name = patp,
                        ports = {
                            '80/tcp':config['http_port'],
                            '34343/udp':config['ames_port']
                            },
                        mounts = [mount],
                        detach=True)

            if c:
                Log.log(f"{patp}: Successfully built container")
                return True
            else:
                raise Exception("Container wasn't created")

        except Exception as e:
            Log.log(f"{patp}: Failed to build container: {e}")
            return False
