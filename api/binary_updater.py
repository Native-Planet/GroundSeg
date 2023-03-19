# Python
import os
import requests
from time import sleep

# GroundSeg modules
from log import Log
from utils import Utils

class BinUpdater:
    def __init__(self, config, debug_mode):
        self.config_object = config
        self.config = config.config
        self.arch = config._arch
        self.base_path = config.base_path
        self.debug_mode = debug_mode

    def check_bin_update(self):
        Log.log("Updater: Binary updater thread started")
        Log.log(f"Updater: Update mode: {self.config['updateMode']}")
        while True:
            try:
                Log.log("Updater: Fetching version server for updated information")
                url = self.config['updateUrl']
                r = requests.get(url)

                if r.status_code == 200:
                    self.config_object.update_avail = True
                    self.config_object.update_payload = r.json()

                    # Run binary update check
                    self.run_check()
                    sleep(self.config['updateInterval'])

                else:
                    raise ValueError(f"Status code {r.status_code}")

            except Exception as e:
                self.config_object.update_avail = False
                Log.log(f"Updater: Unable to retrieve update information: {e}")
                sleep(60)

    def run_check(self):
        try:
            cur_hash = self.config['binHash']
            branch = self.config['updateBranch']
            mode = self.config['updateMode']

            if mode == 'auto':
                # Remove prior failed download
                self.remove_file('groundseg_new')

                # Get payload information
                d = self.config_object.update_payload['groundseg'][branch]['groundseg']

                # Get version
                ver = f"v{d['major']}.{d['minor']}.{d['patch']}"
                old_ver = self.config['gsVersion']
                if branch == 'edge':
                    ver = f"{ver}-edge"
                    old_ver = f"{old_ver}-edge"

                # Show versions
                Log.log(f"Updater: Current {old_ver} | Latest {ver}")

                # Download new version
                dl_hash = d[f'{self.arch}_sha256']
                if cur_hash == dl_hash:
                    Log.log("Updater: No binary update required")
                else:
                    Log.log(f"Updater: Downloading new groundseg binary")

                    # Stream chunks and write to file
                    dl = d[f"{self.arch}_url"]
                    Log.log(f"Updater: Download URL: {dl}")
                    r = requests.get(dl)
                    f = open(f"{self.base_path}/groundseg_new", 'wb')
                    for chunk in r.iter_content(chunk_size=512 * 1024):
                        if chunk:
                            f.write(chunk)
                    f.close()

                    # Check new binary hash
                    new_hash = Utils.make_hash(f"{self.base_path}/groundseg_new")
                    Log.log(f"Updater: Version server binary hash: {dl_hash}")
                    Log.log(f"Updater: Downloaded binary hash: {new_hash}")
                    if new_hash != dl_hash:
                        Log.log(f"Updater: Hash mismatched. Incorrect file downloaded")
                    else:
                        # Remove old binary
                        Log.log("Updater: Removing old groundseg binary")
                        self.remove_file(f"{self.base_path}/groundseg")

                        # Rename new binary
                        Log.log("Updater: Renaming new groundseg binary")
                        self.rename_file(f"{self.base_path}/groundseg_new",
                                         f"{self.base_path}/groundseg")

                        # Make binary executable
                        Log.log("Updater: Setting launch permissions for new binary")
                        os.system(f"chmod +x {self.base_path}/groundseg")

                        # Pause
                        sleep(1)

                        # Restart GroundSeg
                        if self.debug_mode:
                            Log.log("Updater: Debug mode: Skipping restart")
                            Log.log("Updater: Debug mode: Setting new bin hash")
                            self.config['binHash'] = Utils.make_hash(f"{self.base_path}/groundseg")
                            self.config_object.save_config()
                        else:
                            Log.log("Updater: Restarting groundseg...")
                            os.system("systemctl restart groundseg")

        except Exception as e:
            Log.log(f"Updater: Binary updater failed: {e}")


    # Remove file
    def remove_file(self, file):
        try:
            os.remove(file)
        except:
            pass

        while os.path.isfile(file):
            sleep(0.1)

        return True

    # Rename file
    def rename_file(self, old, new):
        os.rename(old,new)

        while not os.path.isfile(new):
            sleep(0.1)

        return True
