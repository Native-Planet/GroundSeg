import os
import asyncio
import requests
from time import sleep

from lib.helper import Helper

class BinUpdate:
    def __init__(self, cfg, base, dev):
        super().__init__()
        self.cfg = cfg
        self.arch = self.cfg.arch
        self.base = base
        self.dev = dev

    async def run(self):
        while True:
            try:
                cur_hash = self.cfg.system.get('binHash')
                branch = self.cfg.system.get('updateBranch')
                mode = self.cfg.system.get('updateMode')

                if mode == 'auto':
                    # Remove prior failed download
                    self.remove_file('groundseg_new')

                # Get payload information
                if not self.cfg.version_server_ready:
                    raise Exception("version server not ready")
                d = self.cfg.version_info.get('groundseg').get(branch).get('groundseg')

                # Get version
                ver = f"v{d['major']}.{d['minor']}.{d['patch']}"
                old_ver = self.cfg.system.get('gsVersion')
                if branch != 'latest':
                    ver = f"{ver}-{branch}"
                    old_ver = f"{old_ver}-{branch}"

                # Show versions
                print(f"updater:binary:run: Current {old_ver} | Latest {ver}")

                # Download new version
                dl_hash = d[f'{self.arch}_sha256']
                if cur_hash == dl_hash:
                    print("updater:binary:run: No binary update required")
                else:
                    print("updater:binary:run: Downloading new groundseg binary")

                    # Stream chunks and write to file
                    dl = d[f"{self.arch}_url"]
                    print(f"updater:binary:run: Download URL: {dl}")
                    r = requests.get(dl)
                    f = open(f"{self.base}/groundseg_new", 'wb')
                    for chunk in r.iter_content(chunk_size=512 * 1024):
                        if chunk:
                            f.write(chunk)
                    f.close()

                    # Check new binary hash
                    new_hash = Helper().make_hash(f"{self.base}/groundseg_new")
                    print(f"updater:binary:run: Version server binary hash: {dl_hash}")
                    print(f"updater:binary:run: Downloaded binary hash: {new_hash}")
                    if new_hash != dl_hash:
                        print("updater:binary:run: Hash mismatched. Incorrect file downloaded")
                    else:
                        # Remove old binary
                        print("updater:binary:run: Removing old groundseg binary")
                        self.remove_file(f"{self.base}/groundseg")

                        # Rename new binary
                        print("updater:binary:run: Renaming new groundseg binary")
                        self.rename_file(f"{self.base}/groundseg_new",
                        f"{self.base}/groundseg")

                        # Make binary executable
                        print("updater:binary:run: Setting launch permissions for new binary")
                        os.system(f"chmod +x {self.base}/groundseg")

                        # Pause
                        sleep(1)

                        # Restart GroundSeg
                        if self.dev:
                            print("updater:binary:run: Dev mode: Skipping restart")
                            print("updater:binary:run: Dev mode: Setting new bin hash")
                            self.cfg.system['binHash'] = Helper().make_hash(f"{self.base}/groundseg")
                            self.cfg.save_config()
                        else:
                            print("updater:binary:run: Restarting groundseg...")
                            os.system("systemctl restart groundseg")

            except Exception as e:
                print(f"updater:binary:run: Binary updater failed: {e}")
            await asyncio.sleep(90)


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
