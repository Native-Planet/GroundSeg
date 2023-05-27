# Python
from time import sleep
from datetime import datetime

# GroundSeg module
from log import Log

class Melder:
    def __init__(self, config, orchestrator):
        self.config_object = config
        self.config = config.config
        self.orchestrator = orchestrator

    # Checks if a meld is due, runs meld
    def meld_loop(self): 
        Log.log("Melder: Meld thread started")
        while True:
            try:
                copied = self.orchestrator.urbit._urbits
                for p in list(copied):
                    try:
                        now = int(datetime.utcnow().timestamp())
                        if copied[p]['meld_schedule']:
                            if int(copied[p]['meld_next']) <= now:
                                self.orchestrator.urbit.send_pack_meld(p)
                    except Exception as e:
                        Log.log(f"Melder: Unable to check meld status of {p}: {e}")

            except Exception as e:
                Log.log(f"Melder: Meld loop error: {e}")

            sleep(30)
