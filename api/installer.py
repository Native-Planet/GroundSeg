import sys
import subprocess
import platform
import urllib.request

from utils import Log


class Installer:

    env_ready = False
    system = platform.system()

    def __init__(self):
        Log.log_groundseg("Checking system for dependencies")
        # Stop and remove legacy containers
        try:
            subprocess.run(['docker', 'rm', '-f', 'groundseg_api', 'groundseg_webui'])
        except:
            pass

        if self.system == 'Linux':
            self.env_ready = self.has_docker()

        else:
            self.env_ready = False

    def has_docker(self):
        if self.system == 'Linux':
            try:
                subprocess.run('docker')
                return True

            except:
                Log.log_groundseg("Docker not installed!")
                Log.log_groundseg("Trying to install Docker")

                try:
                    urllib.request.urlretrieve("https://get.docker.com", "docker_install.sh")
                    subprocess.run(['chmod', '+x', 'docker_install.sh'])
                    subprocess.run(['bash', 'docker_install.sh'])

                except Exception as e:
                    Log.log_groundseg(e)
                    return False

            try:
                subprocess.run(['rm', 'docker_install.sh'])
                subprocess.run('docker')
                return True

            except:
                Log.log_groundseg("Docker failed to install. Please try installing it manually")
                pass

        return False
