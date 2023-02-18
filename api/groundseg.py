from threading import Thread

from flask import Flask
from flask_cors import CORS

from config import Config
from log import Log
from binary_updater import BinUpdater

# Announce
Log.log("---------- Starting GroundSeg ----------")
Log.log("----------- Urbit is love <3 -----------")

# Setup System Config
base_path = "/opt/nativeplanet/groundseg"
system_config = Config(base_path)

# Create flask app
app = Flask(__name__, static_folder=f'{base_path}/static')
CORS(app, supports_credentials=True)

# Initialize Binary Updater class
bin_update = BinUpdater(system_config)

# Check C2C
if False: #temp
    print("c2c mode")
    #C2C stuff
else:
    #Binary Updater
    Thread(target=bin_update.check_bin_updates, daemon=True).start()

# Flask
if __name__ == '__main__':
    port = 27016
    #if orchestrator._c2c_mode:
    if False: #temp
        port = 80
    debug_mode = True
    app.run(host='0.0.0.0', port=port, threaded=True, debug=debug_mode, use_reloader=debug_mode)
