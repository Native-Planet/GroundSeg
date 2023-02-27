import docker
from log import Log

client = docker.from_env()

class MCDocker:
    def start(self, name, updater_info, arch):
        sha = f"{arch}_sha256"
        image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"

        Log.log("MC: Attempting to start container")
        c = self.get_container(name)
        if not c:
            c = self.create_container(name, image)
            if not c:
                return False

        if c.attrs['Config']['Image'] != image:
            Log.log("MC: Container and config versions are mismatched")
            if self.remove_container(name):
                c = self.create_container(name, image)
                if not c:
                    return False

        if c.status == "running":
            Log.log("MC: Container already started")
            return True

        try:
            c.start()
            Log.log("MC: Successfully started container")
            return True
        except:
            Log.log("MC: Failed to start container")
            return False

    def get_container(self, name):
        try:
            c = client.containers.get(name)
            return c
        except:
            Log.log("MC: Container not found")
            return False

    def create_container(self, name, image):
        Log.log("MC: Attempting to create container")
        if self.pull_image(image):
            return self.build_container(name, image)

    def pull_image(self, image):
        try:
            Log.log(f"MC: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"MC: Failed to pull {image}: {e}")
            return False

    def build_container(self, name, image):
        try:
            Log.log("MC: Building container")
            c = client.containers.create(
                    image= image,
                    network='container:wireguard',
                    entrypoint='/bin/bash',
                    stdin_open=True,
                    name = name,
                    detach=True)
            return c

        except Exception as e:
            Log.log(f"MC: Failed to build container: {e}")
            return False

    def remove_container(self, name):
        Log.log("MC: Attempting to remove container")
        c = self.get_container(name)
        if not c:
            Log.log("MC: Failed to remove container")
            return False
        else:
            c.remove(force=True)
            Log.log("MC: Successfully removed container")
            return True

    def exec(self, name, command):
        c = self.get_container(name)
        if c:
            try:
                c.exec_run(command)
                return True
            except Exception as e:
                Log.log(f"{name}: Unable to exec {command}: {e}")

        return False

    '''
    def make_service_account(self, patp, acc, pwd):
        x = None

        print('Updating service account credentials', file=sys.stderr)
        x = self.container.exec_run(f"mc admin user svcacct edit \
                --secret-key '{pwd}' \
                patp_{patp} {acc}", tty=True).output.decode('utf-8').strip()

        if 'ERROR' in x:
            print('Service account does not exist. Creating new account...', file=sys.stderr)
            x = self.container.exec_run(f"mc admin user svcacct add \
                    --access-key '{acc}' \
                    --secret-key '{pwd}' \
                    patp_{patp} {patp}").output.decode('utf-8').strip()
        
            if 'ERROR' in x:
                return 400

        return 200
        
    '''
