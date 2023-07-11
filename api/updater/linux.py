import asyncio
import schedule
import subprocess

class LinUpdate:
    def __init__(self, cfg, dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    async def run(self):
        print("updater:linux:run: Linux updater thread started")
        self.loop()

        val = self.cfg.system.get('linuxUpdates').get('value')
        interval = self.cfg.system.get('linuxUpdates').get('interval')

        if interval == 'week':
            schedule.every(val).weeks.do(self.loop)

        if interval == 'day':
            schedule.every(val).days.do(self.loop)

        if interval == 'hour':
            schedule.every(val).hours.do(self.loop)

        if interval == 'minute':
            schedule.every(val).minutes.do(self.loop)

        print(f"updater:linux:run: Linux updates scheduled for every {val} {interval}{'s' if val > 1 else ''}")

        while True:
            schedule.run_pending()
            await asyncio.sleep(1)

    def loop(self):
        if self.cfg.system.get('updateMode') == 'auto':
            try:
                print("updater:linux:loop: Checking for linux updates")
                if self.dev:
                    # Fake values
                    upgrade, new, remove, ignore = [5,0,0,0]
                else:
                    # Default values
                    upgrade, new, remove, ignore = [0,0,0,0]

                    # Update package list
                    try:
                        print("updater:linux:loop: Running apt update")
                        subprocess.run(['apt','update'])
                    except Exception as e:
                        print(f"updater:linux:loop: Failed to run apt update: {e}")

                    # Simulate upgrade
                    print("updater:linux:loop: Running apt upgrade -s to simulate update")
                    sim_upgrade = ["apt", "upgrade", "-s"]
                    try:
                        sim_res = subprocess.run(sim_upgrade, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                        print(f"sim_res stdout: {sim_res.stdout}")
                    except Exception as e:
                        print(f"updater:linux:loop: Failed to run apt upgrade -s: {e}")

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

                '''
                self.ws_util.system_broadcast('updates', 'linux', 'update', state)
                self.ws_util.system_broadcast('updates', 'linux', 'upgrade', upgrade)
                self.ws_util.system_broadcast('updates', 'linux', 'new', new)
                self.ws_util.system_broadcast('updates', 'linux', 'remove', remove)
                self.ws_util.system_broadcast('updates', 'linux', 'ignore', ignore)
                '''

                print(f"updater:linux:loop: Linux updates: {upgrade} to upgrade, {new} to install, {remove} to remove, {ignore} to ignore")
            except Exception as e:
                print(f"updater:linux:loop: Failed to check for linux updates: {e}")
