import re
import subprocess
from time import sleep

from log import Log

class LinuxUpdater:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.debug_mode = config.debug_mode

    def updater_loop(self):
        Log.log("Updater: Linux updater thread started")
        while True:
            try:
                Log.log("Updater: Checking for linux updates")
                if self.debug_mode:
                    # Fake values
                    upgrade, new, remove, ignore = [5,0,0,0]
                else:
                    # Default values
                    upgrade, new, remove, ignore = [0,0,0,0]

                    # Update package list
                    try:
                        Log.log("Updater: Running apt update")
                        subprocess.run(['apt','update'])
                    except Exception as e:
                        Log.log(f"Updater: Failed to run apt update: {e}")

                    # Simulate upgrade
                    Log.log("Updater: Running apt upgrade -s to simulate update")
                    sim_upgrade = ["apt", "upgrade", "-s"]
                    try:
                        sim_res = subprocess.run(sim_upgrade, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                        Log.log(f"sim_res stdout: {sim_res.stdout}")
                    except Exception as e:
                        Log.log(f"Updater: Failed to run apt upgrade -s: {e}")

                    for ln in sim_res.stdout.split("\n"):
                        pattern = r"(\d+) upgraded, (\d+) newly installed, (\d+) to remove and (\d+) not upgraded."
                        updates = re.match(pattern, ln)
                        if updates:
                            upgrade, new, remove, ignore = map(int, updates.groups())

                # Set update notification
                self.config_object.linux_updates = (upgrade + new + remove) > 0

                Log.log(f"Updater: Linux updates: {upgrade} to upgrade, {new} to install, {remove} to remove, {ignore} to ignore")
            except Exception as e:
                Log.log(f"Updater: Failed to check for linux updates: {e}")

            # Set check interval -- defaults to 48 hours
            sleep(self.config['linuxUpdates'])
