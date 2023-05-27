import os
import time
from log import Log

class KillSwitch:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.debug_mode = self.config_object.debug_mode

    def kill_switch(self):
        interval = self.config['c2cInterval']
        if interval != 0:
            Log.log(f"C2C: Connect to connect interval detected! Restarting device in {interval} seconds")
            time.sleep(self.config['c2cInterval'])

            if debug_mode:
                Log.log("C2C: Debug mode: Skipping device restart")
            else:
                Log.log("C2C: Restarting device")
                os.system("reboot")
        else:
            Log.log("C2C: Connect to connect interval not set!")
