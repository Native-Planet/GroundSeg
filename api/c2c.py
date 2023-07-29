# Python
import os
import sys
import time
import subprocess

# Flask
from flask import Flask, request, jsonify
from flask_cors import CORS

#Modules
import nmcli
from PyAccessPoint import pyaccesspoint

# GroundSeg modules
import html_templates
from lib.wifi import get_wifi_device, list_wifi_ssids, wifi_connect

# Create flask app
class C2C:
    def __init__(self, cfg, dev):
        self.cfg = cfg
        self.dev = dev
        self.wifi_device = get_wifi_device()
        self.static_dir = f"{self.cfg.base}/static"
        self.ssids = None

        self.app = Flask(__name__, static_folder=self.static_dir)
        CORS(self.app, supports_credentials=True)

        '''
        os.system(f"mkdir -p {self.static_dir}")

        # background.png
        if not os.path.isfile(f"{self.static_dir}/background.png"):
            if not static_files.make_if_valid("background.png"):
                print("C2C: Failed to create background.png")
            else:
                print("C2C: Created background.png")

        # nplogo.svg
        if not os.path.isfile(f"{self.static_dir}/nplogo.svg"):
            if not static_files.make_if_valid("nplogo.svg"):
                print("C2C: Failed to create nplogo.svg")
            else:
                print("C2C: Created nplogo.svg")
            
        # Inter-SemiBold.otf
        if not os.path.isfile(f"{self.static_dir}/Inter-SemiBold.otf"):
            if not static_files.make_if_valid("Inter-SemiBold.otf"):
                print("C2C: Failed to create Inter-SemiBold.otf")
            else:
                print("C2C: Created Inter-SemiBold.otf")
        '''

        if not self.dev:
            self.ap = pyaccesspoint.AccessPoint(wlan=self.wifi_device,
                                                ssid='NativePlanet_c2c',
                                                password='nativeplanet')
            self.start_c2c()
        else:
            self.ssids = ['example 1','nativeplanet','not native planet']

        #
        #   Routes
        #


        # Home Page
        @self.app.route("/", methods=['GET','POST'])
        def c2c():
            if request.method == 'GET':
                return html_templates.home_page(self.ssids)
            if request.method == 'POST':
                print("C2C: Manual restart requested. Restarting device")
                os.system("reboot")
                return jsonify(200)
            return jsonify(404)

        # Connect to SSID
        @self.app.route("/connect/<ssid>", methods=['GET','POST'])
        def c2c_ssid(ssid=None):
            if request.method == 'GET':
                return html_templates.connect_page(ssid)

            if request.method == 'POST':
                print(f"C2C: Requested to connect to SSID: {ssid}")
                print("C2C: Turning off Access Point")
                # turn off ap
                try:
                    if self.ap.stop():
                        print("C2C: Starting systemd-resolved")
                        x = subprocess.check_output("systemctl start systemd-resolved", shell=True)
                        if x.decode('utf-8') == '':
                            nmcli.radio.wifi_on()
                            wifi_on = nmcli.radio.wifi()

                            while not wifi_on:
                                print("C2C: Wireless adapter not turned on yet. Trying again..")
                                nmcli.radio.wifi.on()
                                time.sleep(1)
                                wifi_on = nmcli.radio.wifi()

                            time.sleep(1)
                            print("C2C: Scanning for available SSIDs")
                            nmcli.device.wifi_rescan()
                            time.sleep(8)
                            print(f"C2C: Available SSIDs: {list_wifi_ssids()}")

                            completed = wifi_connect(ssid, request.form['password'])
                            if completed and self.cfg.system.get('c2cInterval') == 0:
                                print("C2C: Setting c2c interval to 600 seconds")
                                self.cfg.set_c2c_interval(600)

                            if self.dev:
                                print("C2C: Debug mode: Skipping restart")
                            else:
                                print("C2C: Restarting device..")
                                os.system("reboot")

                            return jsonify(200)

                except Exception as e:
                    print(f"C2C: An error has occurred: {e}")
                    if self.dev:
                        print("C2C: Debug mode: Skipping restart")
                    else:
                        print("C2C: Restarting device..")
                        os.system("reboot")

            return jsonify(404)

    def start_c2c(self):
        try:
            print(f"C2C: Turning wireless adapter {self.wifi_device} on")
            nmcli.radio.wifi_on()
            wifi_on = nmcli.radio.wifi()
            while not wifi_on:
                print("C2C: Wireless adapter not turned on yet. Trying again..")
                nmcli.radio.wifi.on()
                time.sleep(1)
                wifi_on = nmcli.radio.wifi()

            time.sleep(1)
            print("C2C: Scanning for available SSIDs")
            nmcli.device.wifi_rescan()
            time.sleep(8)

            self.ssids = list_wifi_ssids()

            if len(self.ssids) < 1:
                print("C2C: No SSIDs available, exiting..")
                sys.exit()

            print(f"C2C: Available SSIDs: {self.ssids}")
            print("C2C: Stopping systemd-resolved")
            x = subprocess.check_output("systemctl stop systemd-resolved", shell=True)
            if x.decode('utf-8') == '':
                if self.ap.stop():
                    if self.ap.start():
                        print("C2C: Access Point enabled")
                    else:
                        print("C2C: Unable to start Access Point. Exiting..")
                        sys.exit()
                else:
                    print("C2C: Something went wrong. Exiting..")
                    sys.exit()
        except Exception as e:
            print(f"C2C: Connect to connect error: {e}")
            print("C2C: Exiting..")
            sys.exit()


    # Kill port for C2C
    def kill_process(self, port):
        process = False
        print(f"C2C: Finding for process listening on port {port}")

        try:
            output = subprocess.check_output(["lsof", "-i", f"tcp:{port}"])
            pid = int(output.split()[10])
            process = True
        except subprocess.CalledProcessError:
            print(f"C2C: No process is listening on port {port}")
            return True

        if process:
            print("C2C: Attempting to kill process")
            try:
                os.kill(pid, 9)
                print(f"C2C: Process {pid} has been killed")
                return True
            except OSError:
                print(f"C2C: Failed to kill process {pid} on port {port}")
                return False

    # Run Flask app
    def run(self):
        port = 80
        if self.kill_process(port):
            print("C2C: Starting Flask server")
            debug_mode = self.dev
            self.app.run(host='0.0.0.0', port=port, threaded=True, debug=debug_mode, use_reloader=False)
        else:
            print(f"C2C: Port {port} is used! Cannot start Flask server")
