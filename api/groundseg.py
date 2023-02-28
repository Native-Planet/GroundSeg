dev = False
dev = True

# GroundSeg modules
from config import Config
from log import Log
from utils import Utils
from orchestrator import Orchestrator

# Flask apps
from groundseg_flask import GroundSeg
from c2c_flask import C2C

# Threads
from threading import Thread
from system_monitor import SysMonitor
from melder import Melder
from anchor_information import AnchorUpdater
from wireguard_refresher import WireguardRefresher

# Setup System Config
base_path = "/opt/nativeplanet/groundseg"
sys_config = Config(base_path, dev)

# Start Updater
Thread(target=Utils.get_version_info, args=(sys_config, sys_config.debug_mode), daemon=True).start()

# Check C2C
if sys_config.device_mode == "c2c":
    # start c2c kill switch
    print("c2c mode")

    # Flask
    c2c = C2C(sys_config)
    c2c.run()

else:
    # System monitoring
    sys_mon = SysMonitor(sys_config)
    Thread(target=sys_mon.sys_monitor, daemon=True).start()

    # Start GroundSeg orchestrator
    orchestrator = Orchestrator(sys_config)

    # Scheduled melds
    meld_loop = Melder(sys_config, orchestrator)
    Thread(target=meld_loop.meld_loop, daemon=True).start()

    # Anchor information
    anchor_loop = AnchorUpdater(sys_config, orchestrator)
    Thread(target=anchor_loop.anchor_loop, daemon=True).start()

    # Wireguard connection refresher
    wg_refresher = WireguardRefresher(sys_config, orchestrator)
    Thread(target=wg_refresher.refresh_loop, daemon=True).start()

    # Flask
    groundseg = GroundSeg(sys_config, orchestrator)
    groundseg.run()
