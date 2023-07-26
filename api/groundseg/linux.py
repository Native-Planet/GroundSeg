import subprocess
from time import sleep

class UpdateLinux:
    def __init__(self,cfg,dev):
        self.cfg = cfg
        self.dev = dev

    def run_update(self):
        try:
            print(f"groundseg:update_linux:run_update: Attempting to run upgrade command")
            cmd = ["apt", "upgrade", "-y"]
            try:
                if self.dev:
                    output = True
                    print("groundseg:update_linux:run_update: Skipping apt upgrade command in debug mode")
                    sleep(3)
                else:
                    result = subprocess.run(cmd, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                    output = result.stdout

                if output:
                    self.cfg.system['linuxUpdates']['previous'] = True
                    self.cfg.save_config()
                    print(f"groundseg:update_linux:run_update: Updates has been installed. Restarting")
                    sleep(1)

                if self.dev:
                    print("groundseg:update_linux:run_update: Skipping device restart in debug mode")
                else:
                    subprocess.run('reboot')

            except Exception as e:
                raise Exception(e)
        except Exception as e:
            print(f"groundseg:update_linux:run_update:: Linux update failed: {e}")
