# Python
import time
import requests

# GroundSeg modules
from log import Log

class WireguardRefresher:
    def __init__(self, config, orchestrator):
        self.config_object = config
        self.config = config.config
        self.orchestrator = orchestrator
        self.wireguard = self.orchestrator.wireguard
        self.urbit = self.orchestrator.urbit
        self.minio = self.orchestrator.minio
        self.failed = []

    # Checks if wireguard connection is functional, restarts wireguard
    def refresh_loop(self):
        Log.log("wireguard_refresher:refresh_loop Thread started")
        count = 0
        while True:
            if count > 1:
                count = 0
                self.failed = []
            try:
                if self.config['wgOn'] and self.config_object.anchor_ready:
                    copied = self.urbit._urbits
                    for p in list(copied):
                        running = False

                        c = self.urbit.urb_docker.get_container(p)
                        if c:
                            running = c.status == "running"
                        if running and copied[p]['network'] != "none":
                            res = requests.get(f"https://{copied[p]['wg_url']}/~_~/healthz")
                            if res.status_code == 502:
                                if self.failure_check(p):
                                    Log.log(f"wireguard_refresher:refresh_loop Failed: {self.failed}")
                                    Log.log("wireguard_refresher:refresh_loop StarTram connection is broken. Restarting")
                                    self.failed = []
                                    self.orchestrator.startram_restart()
                                    break

            except Exception as e:
                Log.log(f"WG Refresher: {e}")

            count += 1
            time.sleep(60)

    def failure_check(self, p):
        if p in self.failed:
            return True
        else:
            self.failed.append(p)
            return False

