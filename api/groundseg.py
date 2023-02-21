# Flask
from flask import Flask
from flask_cors import CORS

# GroundSeg modules
from config import Config
from log import Log
from utils import Utils
from orchestrator import Orchestrator

# Threads
from threading import Thread
from system_monitor import SysMonitor

# Dev
debug_mode = True

# Announce
if debug_mode:
    Log.log("---------- Starting GroundSeg in debug mode ----------")
else:
    Log.log("----------------- Starting GroundSeg -----------------")
    Log.log("------------------ Urbit is love <3 ------------------")

# Setup System Config
base_path = "/opt/nativeplanet/groundseg"
system_config = Config(base_path)

# Create flask app
app = Flask(__name__, static_folder=f'{base_path}/static')
CORS(app, supports_credentials=True)

# Start Updater
Thread(target=Utils.get_version_info, args=(system_config, debug_mode), daemon=True).start()

# Check C2C
if system_config.device_mode == "c2c":
    # start c2c kill switch
    print("c2c mode")
else:
    # Start GroundSeg orchestrator
    orchestrator = Orchestrator(system_config)

    # System monitoring
    sys_mon = SysMonitor(system_config)
    Thread(target=sys_mon.sys_monitor, daemon=True).start()

    # Meld loop

    # Anchor information

    # Wireguard connection refresher

# Flask
if __name__ == '__main__':
    port = 27016
    #if orchestrator._c2c_mode:
    if False: #temp
        port = 80
    app.run(host='0.0.0.0', port=port, threaded=True, debug=debug_mode, use_reloader=debug_mode)
