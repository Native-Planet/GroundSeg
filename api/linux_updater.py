import re
import subprocess
from time import sleep

import schedule

from log import Log

class LinuxUpdater:
    def __init__(self, config, ws_util):
        self.config_object = config
        self.config = config.config
        self.debug_mode = config.debug_mode
        self.ws_util = ws_util

    def run(self):
        Log.log("linux-updater: Linux updater thread started")
        self.updater_loop()

        val = self.config['linuxUpdates']['value']
        interval = self.config['linuxUpdates']['interval']

        if interval == 'week':
            schedule.every(val).weeks.do(self.updater_loop)

        if interval == 'day':
            schedule.every(val).days.do(self.updater_loop)

        if interval == 'hour':
            schedule.every(val).hours.do(self.updater_loop)

        if interval == 'minute':
            schedule.every(val).minutes.do(self.updater_loop)

        Log.log(f"linux-updater: Linux updates scheduled for every {val} {interval}{'s' if val > 1 else ''}")

        while True:
            schedule.run_pending()
            sleep(1)

    def updater_loop(self):
        if self.config['updateMode'] == 'auto':
            try:
                Log.log("linux-updater: Checking for linux updates")
                if self.debug_mode:
                    # Fake values
                    upgrade, new, remove, ignore = [5,0,0,0]
                else:
                    # Default values
                    upgrade, new, remove, ignore = [0,0,0,0]

                    # Update package list
                    try:
                        Log.log("linux-updater: Running apt update")
                        subprocess.run(['apt','update'])
                    except Exception as e:
                        Log.log(f"linux-updater: Failed to run apt update: {e}")

                    # Simulate upgrade
                    Log.log("linux-updater: Running apt upgrade -s to simulate update")
                    sim_upgrade = ["apt", "upgrade", "-s"]
                    try:
                        sim_res = subprocess.run(sim_upgrade, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                        Log.log(f"sim_res stdout: {sim_res.stdout}")
                    except Exception as e:
                        Log.log(f"linux-updater: Failed to run apt upgrade -s: {e}")

                    for ln in sim_res.stdout.split("\n"):
                        pattern = r"(\d+) upgraded, (\d+) newly installed, (\d+) to remove and (\d+) not upgraded."
                        updates = re.match(pattern, ln)
                        if updates:
                            upgrade, new, remove, ignore = map(int, updates.groups())
                            break

                # Set update notification
                state = 'updated'
                if (upgrade + new + remove) > 0:
                    state = 'pending'

                self.ws_util.system_broadcast('updates', 'linux', 'update', state)
                self.ws_util.system_broadcast('updates', 'linux', 'upgrade', upgrade)
                self.ws_util.system_broadcast('updates', 'linux', 'new', new)
                self.ws_util.system_broadcast('updates', 'linux', 'remove', remove)
                self.ws_util.system_broadcast('updates', 'linux', 'ignore', ignore)

                Log.log(f"linux-updater: Linux updates: {upgrade} to upgrade, {new} to install, {remove} to remove, {ignore} to ignore")
            except Exception as e:
                Log.log(f"linux-updater: Failed to check for linux updates: {e}")
