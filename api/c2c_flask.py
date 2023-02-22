# Python
import os
import subprocess

# Flask
from flask import Flask
from flask_cors import CORS

# GroundSeg modules
from log import Log

# Create flask app
class C2C:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config

        self.app = Flask(__name__, static_folder=f'{self.config_object.base_path}/static')
        CORS(self.app, supports_credentials=True)

        #
        #   Routes
        #


        # Home Page
        #@self.app.route("/", methods=['GET','POST'])

        # Connect to SSID
        #@self.app.route("/connect/<ssid>", methods=['GET','POST'])

    # Kill port for C2C
    def kill_process(self, port):
        process = False

        Log.log(f"C2C: Finding for process listening on port {port}")
        try:
            output = subprocess.check_output(["lsof", "-i", f"tcp:{port}"])
            pid = int(output.split()[10])
            process = True
        except subprocess.CalledProcessError:
            Log.log(f"C2C: No process is listening on port {port}")
            return True

        if process:
            Log.log("C2C: Attempting to kill process")
            try:
                os.kill(pid, 9)
                Log.log(f"C2C: Process {pid} has been killed")
                return True
            except OSError:
                Log.log(f"C2C: Failed to kill process {pid} on port {port}")
                return False

    # Run Flask app
    def run(self):
        port = 80
        if self.kill_process(port):
            Log.log("C2C: Starting Flask server")
            debug_mode = self.config_object.debug_mode
            self.app.run(host='0.0.0.0', port=port, threaded=True, debug=debug_mode, use_reloader=debug_mode)
        else:
            Log.log(f"C2C: Port {port} is used! Cannot start Flask server")
