import os
import requests

from time import sleep

from log import Log

class BinUpdater:

    def check_bin_update(self, config, debug_mode):

        try:

            cur_hash = config.config['binHash']
            branch = config.config['updateBranch']
            mode = config.config['updateMode']

            if mode == 'auto':
                # Remove prior failed download
                self.remove_file('groundseg_new')

                # Get payload information
                d = config.update_payload['groundseg'][branch]['groundseg']

                # Get version
                ver = f"v{d['major']}.{d['minor']}.{d['patch']}"
                if branch == 'edge':
                    ver = f"{ver}-edge"

                # Show versions
                Log.log(f"Updater: Current {config.config['gsVersion']} | Latest {ver}")

                # Download new version
                if cur_hash == d[f'{config._arch}_sha256']:
                    Log.log("Updater: No binary update required")
                else:
                    Log.log(f"Updater: Downloading new groundseg binary")

                    # Stream chunks and write to file
                    dl = d[f"{config._arch}_url"]
                    r = requests.get(dl)
                    f = open(f"{config.base_path}/groundseg_new", 'wb')
                    for chunk in r.iter_content(chunk_size=512 * 1024):
                        if chunk:
                            f.write(chunk)
                    f.close()

                    # Remove old binary
                    Log.log("Updater: Removing old groundseg binary")
                    self.remove_file(f"{config.base_path}/groundseg")

                    # Rename new binary
                    Log.log("Updater: Renaming new groundseg binary")
                    self.rename_file(f"{config.base_path}/groundseg_new",
                                     f"{config.base_path}/groundseg")

                    # Make binary executable
                    Log.log("Updater: Setting launch permissions for new binary")
                    os.system(f"chmod +x {config.base_path}/groundseg")

                    # Pause
                    sleep(1)

                    # Restart GroundSeg
                    if debug_mode:
                        Log.log("Updater: Debug mode: Skipping restart")
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
