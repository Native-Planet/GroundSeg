import os
import requests
import platform

from time import sleep

from log import Log

class BinUpdater:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.branch = self.config['updateBranch']
        self.url = self.config['updateUrl']
        self.mode = self.config['updateMode']

        self.arch = self.get_arch()
        
    # Main Function
    def check_bin_updates(self):
        Log.log("Binary updater thread started")
        cur_hash = self.config['binHash']

        while True:
            try:
                if self.mode == 'auto':
                    # Announce
                    Log.log('Checking for binary updates')

                    # Remove prior failed download
                    self.remove_file('groundseg_new')

                    # Get update blob
                    self.config_object.update_payload = requests.get(self.url).json()
                    d = self.config_object.update_payload['groundseg'][self.branch]['groundseg']

                    # Get version
                    ver = f"v{d['major']}.{d['minor']}.{d['patch']}"
                    if self.branch == 'edge':
                        ver = f"{ver}-edge"

                    # Show versions
                    Log.log(f"Current version: {self.config['gsVersion']}")
                    Log.log(f"Latest version: {ver}")

                    # Download new version
                    if cur_hash == d[f'{self.arch}_sha256']:
                        Log.log("No binary update required")
                    else:
                        Log.log(f"Downloading new groundseg binary")

                        # Stream chunks and write to file
                        dl = d[f"{self.arch}_url"]
                        r = requests.get(dl)
                        f = open(f"{self.config['CFG_DIR']}/groundseg_new", 'wb')
                        for chunk in r.iter_content(chunk_size=512 * 1024):
                            if chunk:
                                f.write(chunk)
                        f.close()

                        # Remove old binary
                        Log.log("Removing old groundseg binary")
                        self.remove_file('groundseg')

                        # Rename new binary
                        Log.log("Renaming new groundseg binary")
                        self.rename_file('groundseg_new','groundseg')

                        # Make binary executable
                        Log.log("Setting launch permissions for new binary")
                        os.system(f"chmod +x {self.config['CFG_DIR']}/groundseg")

                        # Pause
                        sleep(1)

                        # Restart GroundSeg
                        Log.log("Restarting groundseg...")
                        os.system("systemctl restart groundseg")

            except Exception as e:
                Log.log(f"Binary updater failed: {e}")

            sleep(self.config['updateInterval'])
    # Get system architecture
    def get_arch(self):
        arch = "arm64"
        try:
            if platform.machine() == 'x86_64':
                arch = "amd64"
        except:
            Log.log("Unable to get architecture. Defaulting to arm64")

        return arch

    # Remove file
    def remove_file(self, file):
        path = f"{self.config['CFG_DIR']}/{file}"
        try:
            os.remove(path)
        except:
            pass

        while os.path.isfile(path):
            sleep(0.1)

        return True

    # Rename file
    def rename_file(self, old_name, new_name):
        old = f"{self.config['CFG_DIR']}/{old_name}"
        new = f"{self.config['CFG_DIR']}/{new_name}"
        os.rename(old,new)

        while not os.path.isfile(new):
            sleep(0.1)

        return True
