import subprocess
from time import sleep
from log import Log

class LinuxUpdate:
    def __init__(self, ws_util, config):
        self.config_object = config
        self.config = config.config
        self.base_path = config.base_path
        self._debug = config.debug_mode
        self.ws_util = ws_util

    def run(self, old_info):
        try:
            Log.log(f"updates:linux:update Attempting to run upgrade command")
            self.broadcast("command")
            cmd = ["apt", "upgrade", "-y"]
            try:
                if self._debug:
                    output = True
                    Log.log("updates:linux:update Skipping apt upgrade command in debug mode")
                    sleep(3)
                else:
                    result = subprocess.run(cmd, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                    output = result.stdout

                if output:
                    self.config['linuxUpdates']['previous'] = True
                    self.config_object.save_config()
                    self.broadcast("restarting")
                    Log.log(f"updates:linux:update Updates has been installed. Restarting")

                    if self._debug:
                        Log.log("updates:linux:update Skipping device restart in debug mode")
                    else:
                        subprocess.run('reboot')

            except Exception as e:
                raise Exception(e)

        except Exception as e:
            Log.log(f"updates:linux:update: Linux update failed: {e}")
            self.broadcast(f"failure\n{e}")

        if not self._debug:
            sleep(3)
            self.broadcast(old_info)

    def broadcast(self, info):
        return self.ws_util.system_broadcast('updates','linux','update', info)
