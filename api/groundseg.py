import docker
import threading
import time
import os
import copy
import shutil
import psutil
import sys
import requests
import urllib.request
import nmcli
import subprocess
import html_templates

from datetime import datetime
from flask import Flask, jsonify, request, make_response, render_template
from flask_cors import CORS
from werkzeug.utils import secure_filename
from PyAccessPoint import pyaccesspoint

from utils import Log, Network
from orchestrator import Orchestrator

# Create flask app
app = Flask(__name__)
CORS(app, supports_credentials=True)

# Announce
Log.log_groundseg("---------- Starting GroundSeg ----------")
Log.log_groundseg("----------- Urbit is love <3 -----------")

# Load GroundSeg
orchestrator = Orchestrator("/opt/nativeplanet/groundseg/settings/system.json")

ssids = None
ap = pyaccesspoint.AccessPoint(wlan=orchestrator._wifi_device, ssid='NativePlanet_c2c', password='nativeplanet')

# Handle Connect to Connect
if orchestrator._c2c_mode:
    try:
        Log.log_groundseg(f"Turning wireless adapter {orchestrator._wifi_device} on")
        nmcli.radio.wifi_on()
        wifi_on = nmcli.radio.wifi()
        while not wifi_on:
            Log.log_groundseg("Wireless adapter not turned on yet. Trying again..")
            nmcli.radio.wifi.on()
            time.sleep(1)
            wifi_on = nmcli.radio.wifi()

        #time.sleep(1)
        #for c in nmcli.connection():
        #    if c.conn_type == 'wifi':
        #        Log.log_groundseg(f"Deleting {c.name}")
        #        nmcli.connection.delete(c.name)

        time.sleep(1)
        Log.log_groundseg("Scanning for available SSIDs")
        nmcli.device.wifi_rescan()
        time.sleep(8)

        ssids = Network.list_wifi_ssids()

        if len(ssids) < 1:
            Log.log_groundseg("No SSIDs available, exiting..")
            sys.exit()

        Log.log_groundseg(f"Available SSIDs: {ssids}")
        Log.log_groundseg("Stopping systemd-resolved")
        x = subprocess.check_output("systemctl stop systemd-resolved", shell=True)
        if x.decode('utf-8') == '':
            if ap.stop():
                if ap.start():
                    Log.log_groundseg("Access Point enabled")
                else:
                    Log.log_groundseg("Unable to start Access Point. Exiting..")
                    sys.exit()
            else:
                Log.log_groundseg("Something went wrong. Exiting..")
                sys.exit()
    except Exception as e:
        Log.log_groundseg(f"Connect to connect error: {e}")
        Log.log_groundseg("Exiting..")
        sys.exit()

# Docker Updater
def check_docker_updates():
    Log.log_groundseg("Docker updater thread started")

    while True:
        if orchestrator.config['updateMode'] == 'auto':
            webui = f"nativeplanet/groundseg-webui:{orchestrator.config['updateBranch']}"
            urbit = "tloncorp/urbit:latest"
            minio = "quay/minio/minio"

            update_list = [ webui, urbit, minio]

            client = docker.from_env()
            images = client.images.list()

            for i in list(images):
                try:
                    img = i.tags[0]
                    if img in update_list:
                        old_hash = i.id
                        new_hash = client.images.pull(img).id

                        if not old_hash == new_hash:
                            if img == webui:
                                Log.log_groundseg("Updating WebUI")
                                orchestrator.start_webui()

                            if img == urbit:
                                Log.log_groundseg("Updating Urbit")
                                orchestrator.load_urbits()

                            if img == minio:
                                Log.log_groundseg("Updating MinIOs")
                                orchestrator.reload_minios()
                except:
                    pass

        time.sleep(orchestrator.config['updateInterval'])


# Binary Updater
def check_bin_updates():
    Log.log_groundseg("Binary updater thread started")
    cur_hash = orchestrator.config['binHash']

    while True:
        update_branch = orchestrator.config['updateBranch']
        if update_branch == 'latest':
            update_url = 'https://version.infra.native.computer/version.csv'

        if update_branch == 'edge':
            update_url = 'https://version.infra.native.computer/version_edge.csv'

        try:
            new_name, new_hash, dl_url = requests.get(update_url).text.split('\n')[0].split(',')[0:3]

            if orchestrator.config['updateMode'] == 'auto' and cur_hash != new_hash:
                Log.log_groundseg(f"Latest version: {new_name}")
                Log.log_groundseg("Downloading new groundseg binary")

                #urllib.request.urlretrieve(dl_url, f"{orchestrator.config['CFG_DIR']}/groundseg_new")
                r = requests.get(dl_url)
                f = open(f"{orchestrator.config['CFG_DIR']}/groundseg_new", 'wb')
                for chunk in r.iter_content(chunk_size=512 * 1024):
                    if chunk: # filter out keep-alive new chunks
                        f.write(chunk)
                f.close()

                Log.log_groundseg("Removing old groundseg binary")

                try:
                    os.remove(f"{orchestrator.config['CFG_DIR']}/groundseg")
                except:
                    pass

                time.sleep(3)

                Log.log_groundseg("Renaming new groundseg binary")
                os.rename(f"{orchestrator.config['CFG_DIR']}/groundseg_new",
                        f"{orchestrator.config['CFG_DIR']}/groundseg")

                time.sleep(2)
                Log.log_groundseg("Setting launch permissions for new binary")
                os.system(f"chmod +x {orchestrator.config['CFG_DIR']}/groundseg")

                time.sleep(1)

                Log.log_groundseg("Restarting groundseg...")

                if sys.platform == "darwin":
                    os.system("launchctl load /Library/LaunchDaemons/io.nativeplanet.groundseg.plist")
                else:
                    os.system("systemctl restart groundseg")

        except Exception as e:
            Log.log_groundseg(e)

        time.sleep(orchestrator.config['updateInterval'])


# Get updated Anchor information every 12 hours
def anchor_information():
    Log.log_groundseg("Anchor information thread started")
    while True:
        response = None
        if orchestrator.config['wgRegistered']:
            try:
                url = orchestrator.config['endpointUrl']
                pubkey = orchestrator.config['pubkey']
                headers = {"Content-Type": "application/json"}

                response = requests.get(
                        f'https://{url}/v1/retrieve?pubkey={pubkey}',
                        headers=headers).json()
            
                orchestrator.anchor_config = response
                Log.log_groundseg(response)
                time.sleep(60 * 60 * 12)

            except Exception as e:
                Log.log_groundseg(e)
                time.sleep(60)
        else:
            time.sleep(60)

# Constantly update system information
def sys_monitor():
    Log.log_groundseg("System monitor thread started")
    error = False
    while not orchestrator._vm:
        if error:
            Log.log_groundseg("System monitor error, 15 second timeout")
            time.sleep(15)
            error = False

        # RAM info
        try:
            orchestrator._ram = psutil.virtual_memory().percent
        except Exception as e:
            orchestrator._ram = 0.0
            Log.log_groundseg(e)
            error = True

        # CPU info
        try:
            orchestrator._cpu = psutil.cpu_percent(1)
        except Exception as e:
            orchestrator._cpu = 0.0
            Log.log_groundseg(e)
            error = True

        # CPU Temp info
        try:
            orchestrator._core_temp = psutil.sensors_temperatures()['coretemp'][0].current
        except Exception as e:
            orchestrator._core_temp = 0.0
            Log.log_groundseg(e)
            error = True

        # Disk info
        try:
            orchestrator._disk = shutil.disk_usage("/")
        except Exception as e:
            orchestrator._disk = [0,0,0]
            Log.log_groundseg(e)
            error = True


# Checks if a meld is due, runs meld
def meld_loop():
    Log.log_groundseg("Meld thread started")
    while True:
        copied = orchestrator._urbits
        for p in list(copied):
            try:
                now = int(datetime.utcnow().timestamp())

                if copied[p].config['meld_schedule']:
                    if int(copied[p].config['meld_next']) <= now:
                        x = orchestrator.get_urbit_loopback_addr(copied[p].config['pier_name'])
                        copied[p].send_meld(x)
            except:
                break

        time.sleep(30)

def c2c_kill_switch():
    interval = orchestrator.config['c2cInterval']
    if interval != 0:
        Log.log_groundseg(f"Connect to connect interval detected! Restarting device in {interval} seconds"
        time.sleep(orchestrator.config['c2cInterval'])
        os.system("reboot")
    else:
        Log.log_groundseg("Connect to connect interval not set!")

# Threads
if not orchestrator._c2c_mode:
    threading.Thread(target=check_docker_updates).start() # Docker updater
    threading.Thread(target=check_bin_updates).start() # Binary updater
    threading.Thread(target=sys_monitor).start() # System monitoring
    threading.Thread(target=meld_loop).start() # Meld loop
    threading.Thread(target=anchor_information).start() # Anchor information
else:
    threading.Thread(target=c2c_kill_switch).start # Reboot device after delay

#
#   Endpoints
#

# Check if cookie is valid
@app.route("/cookies", methods=['GET'])
def check_cookies():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if sessionid in orchestrator.config['sessions']:
        return jsonify(200)

    return jsonify(404)

# Get all urbits
@app.route("/urbits", methods=['GET'])
def all_urbits():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:

        urbs = orchestrator.get_urbits()
        res = make_response(jsonify(urbs))
        return res

    return jsonify(404)

# Handle urbit ID related requests
@app.route('/urbit', methods=['GET','POST'])
def urbit_info():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    urbit_id = request.args.get('urbit_id')
    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:

        if request.method == 'GET':
            urb = orchestrator.get_urbit(urbit_id)
            return jsonify(urb)

        if request.method == 'POST':
            res = orchestrator.handle_urbit_post_request(urbit_id, request.get_json())
            return orchestrator.custom_jsonify(res)

    return jsonify(404)

# Handle device's system settings
@app.route("/system", methods=['GET','POST'])
def system_settings():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:

        if request.method == 'GET':
            settings = orchestrator.get_system_settings()
            return jsonify(settings)

        if request.method == 'POST':
            module = request.args.get('module')
            res = orchestrator.handle_module_post_request(module, request.get_json(), sessionid)
            return jsonify(res)

    return jsonify(404)

# Handle anchor registration related information
@app.route("/anchor", methods=['GET'])
def anchor_settings():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:

        if request.method == 'GET':
            settings = orchestrator.get_anchor_settings()
            return jsonify(settings)

    return jsonify(404)

@app.route("/bug", methods=['POST'])
def bug_report():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:
        return jsonify(orchestrator.handle_bug_report(request.get_json()))

    return jsonify(404)

# Pier upload
@app.route("/upload", methods=['POST'])
def pier_upload():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:
    
        if orchestrator.config['updateMode'] == 'auto':
            # change to temp mode (DO NOT SAVE CONFIG)
            orchestrator.config['updateMode'] = 'temp'

        # Uploaded pier
        file = request.files['file']
        filename = secure_filename(file.filename)
        patp = filename.split('.')[0]
        
        # Create subfolder
        file_subfolder = f"{orchestrator.config['CFG_DIR']}/uploaded/{patp}"
        os.makedirs(file_subfolder, exist_ok=True)

        fn = save_path = f"{file_subfolder}/{filename}"
        current_chunk = int(request.form['dzchunkindex'])

        if current_chunk == 0:
            try:
                os.remove(save_path)
                Log.log_groundseg("Cleaning up old files")
            except:
                Log.log_groundseg("Directory is clear")
                pass
            
        if os.path.exists(save_path) and current_chunk == 0:
            os.remove(save_path)

            if orchestrator.config['updateMode'] == 'temp':
                orchestrator.config['updateMode'] = 'auto'

            return jsonify("File exists, try again")

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(request.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except Exception as e:
            Log.log_groundseg(e,file=sys.stderr)

            if orchestrator.config['updateMode'] == 'temp':
                orchestrator.config['updateMode'] = 'auto'

            return jsonify("Can't write file")

        total_chunks = int(request.form['dztotalchunkcount'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if os.path.getsize(save_path) != int(request.form['dztotalfilesize']):

                if orchestrator.config['updateMode'] == 'temp':
                    orchestrator.config['updateMode'] = 'auto'

                # size mismatch
                return jsonify("File size mismatched")
            else:
                return jsonify(orchestrator.boot_existing_urbit(filename))
        else:
            # Not final chunk yet
            return jsonify(200)

    if orchestrator.config['updateMode'] == 'temp':
        orchestrator.config['updateMode'] = 'auto'

    return jsonify(404)

# Login
@app.route("/login", methods=['POST'])
def login():
    if orchestrator.config['firstBoot']:
        return jsonify('setup')

    res = orchestrator.handle_login_request(request.get_json())
    if res == 200:
        res = make_response(jsonify(res))
        res.set_cookie('sessionid', orchestrator.make_cookie())
    else:
        res = make_response(jsonify(res))

    return res

# Setup
@app.route("/setup", methods=['POST'])
def setup():
    if not orchestrator.config['firstBoot']:
        return jsonify(400)

    page = request.args.get('page')

    res = orchestrator.handle_setup(page, request.get_json())

    return jsonify(res)

# c2c
@app.route("/", methods=['GET'])
def c2c():
    if orchestrator._c2c_mode:
        return html_templates.home_page(ssids) ##render_template('connect.html', ssids=ssids)
    return jsonify(404)

@app.route("/connect/<ssid>", methods=['GET','POST'])
def c2ssid(ssid=None):
    if orchestrator._c2c_mode:

        if request.method == 'GET':
            return html_templates.connect_page(ssid)

        if request.method == 'POST':
            Log.log_groundseg(f"Requested to connect to SSID: {ssid}")
            Log.log_groundseg(f"Turning off Access Point")

            # turn off ap
            if ap.stop():
                Log.log_groundseg("Starting systemd-resolved")
                x = subprocess.check_output("systemctl start systemd-resolved", shell=True)
                if x.decode('utf-8') == '':
                    nmcli.radio.wifi_on()
                    wifi_on = nmcli.radio.wifi()

                    while not wifi_on:
                        Log.log_groundseg("Wireless adapter not turned on yet. Trying again..")
                        nmcli.radio.wifi.on()
                        time.sleep(1)
                        wifi_on = nmcli.radio.wifi()

                    Log.log_groundseg(f"Available SSIDs: {Network.list_wifi_ssids()}")

                    completed = Network.wifi_connect(ssid, request.form['password'])
                    Log.log_groundseg(f"Connection to wifi network {completed}")

                    if completed == "success" and orchestrator.config['c2cInverval'] == 0:
                        orchestrator.config['c2cInterval'] = 600
                        orchestrator.save_config()

                    os.system("reboot")
                    return jsonify(200)
    return jsonify(404)

@app.route("/connect/reload/page", methods=['GET'])
def c2crestart():
    if orchestrator._c2c_mode:
        Log.log_groundseg("Connect to Connect: Restarting device")
        os.system("reboot")
        return jsonify(200)
    return jsonify(404)

if __name__ == '__main__':
    port = 27016
    if orchestrator._c2c_mode:
        port = 80
    debug_mode = False
    app.run(host='0.0.0.0', port=port, threaded=True, debug=debug_mode, use_reloader=debug_mode)
