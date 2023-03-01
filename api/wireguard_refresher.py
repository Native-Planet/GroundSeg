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

    # Checks if wireguard connection is functional, restarts wireguard
    def refresh_loop(self):
        Log.log("WG Refresher: Thread started")
        while False:
        #while True:
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
                                Log.log("WG Refresher: Anchor connection is broken. Restarting")
                                self.wireguard.restart(self.urbit, self.minio)
                                break
            except Exception as e:
                Log.log(f"WG Refresher: {e}")

            time.sleep(60)
