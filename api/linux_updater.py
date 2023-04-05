import re
import subprocess
from time import sleep

from log import Log

class LinuxUpdater:
    def __init__(self, config):
        self.config_object = config
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
                    subprocess.run(['apt','update'], shell=True)

                    # Simulate upgrade
                    output = subprocess.check_output(['apt','upgrade','-s'], shell=True).decode('utf-8').strip().split('\n')[-1]

                    # Regex
                    pattern = r"(\d+) upgraded, (\d+) newly installed, (\d+) to remove and (\d+) not upgraded."
                    updates = re.match(pattern, output)
                    if updates:
                        upgrade, new, remove, ignore = map(int, updates.groups())

                # Set update notification
                self.config_object.linux_updates = (upgrade + new + remove) > 0

                Log.log(f"Updater: Linux updates: {upgrade} to upgrade, {new} to install, {remove} to remove, {ignore} to ignore")
            except Exception as e:
                Log.log(f"Updater: Failed to check for linux updates: {e}")

            # Check every 6 hours
            sleep(60 * 60 * 6)
