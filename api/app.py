import threading, time, os, zipfile, tarfile, copy, shutil, psutil, sys, requests
from datetime import datetime
from flask import Flask, jsonify, request, make_response
from flask_cors import CORS
from werkzeug.utils import secure_filename
from orchestrator import Orchestrator

# Load GroundSeg
orchestrator = Orchestrator("/settings/system.json")

app = Flask(__name__)
app.config['TEMP_FOLDER'] = '/tmp/'

CORS(app, supports_credentials=True)

# Get updated Anchor information every 12 hours
def anchor_information():
    print("Anchor information thread started", file=sys.stderr)
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
                print(response, file=sys.stderr)
                time.sleep(60 * 60 * 12)

            except Exception as e:
                print(e, file=sys.stderr)
                time.sleep(60)
        else:
            time.sleep(60)

# Constantly update system information
def sys_monitor():
    print("System monitor thread started", file=sys.stderr)
    while True:
        orchestrator._ram = psutil.virtual_memory().percent
        orchestrator._cpu = psutil.cpu_percent(1)
        orchestrator._core_temp = psutil.sensors_temperatures()['coretemp'][0].current
        orchestrator._disk = shutil.disk_usage("/")

# Checks if a meld is due, runs meld
def meld_loop():
    print("Meld thread started", file=sys.stderr)
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

# Start system monitoring on a new thread
threading.Thread(target=sys_monitor).start()

# Start meld loop on a new thread
threading.Thread(target=meld_loop).start()

# Start anchor information loop on a new thread
threading.Thread(target=anchor_information).start()

#
#   Endpoints
#

# Get all urbits

@app.route("/cookies", methods=['GET'])
def check_cookies():
    sessionid = request.args.get('sessionid')

    if sessionid in orchestrator.config['sessions']:
        return jsonify(200)

    return jsonify(404)

@app.route("/urbits", methods=['GET'])
def all_urbits():
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
    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:

        if request.method == 'GET':
            settings = orchestrator.get_anchor_settings()
            return jsonify(settings)


# Pier upload
@app.route("/upload", methods=['POST'])
def pier_upload():
    sessionid = request.args.get('sessionid')

    if len(str(sessionid)) != 64:
        sessionid = request.cookies.get('sessionid')

    if sessionid == None:
        return jsonify(404)

    if sessionid in orchestrator.config['sessions']:
    
        # Uploaded pier
        file = request.files['file']
        filename = secure_filename(file.filename)
        patp = filename.split('.')[0]
        
        # Create subfolder
        file_subfolder = f"{app.config['TEMP_FOLDER']}{patp}"
        os.system(f"mkdir -p {file_subfolder}")

        fn = save_path = f"{app.config['TEMP_FOLDER']}{patp}/{filename}"
        current_chunk = int(request.form['dzchunkindex'])
            
        if os.path.exists(save_path) and current_chunk == 0:
            # 400 and 500s will tell dropzone that an error occurred and show an error
            os.remove(save_path)
            # File already exists
            return jsonify(200)

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(request.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except OSError:
            # log.exception will include the traceback so we can see what's wrong
            # Could not write to file
            return jsonify(500)

        total_chunks = int(request.form['dztotalchunkcount'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if os.path.getsize(save_path) != int(request.form['dztotalfilesize']):
                # size mismatch
                return jsonify(501)
            else:

                # Extract pier
                try:
                    if filename.endswith("zip"):
                        with zipfile.ZipFile(fn) as zip_ref:
                            zip_ref.extractall(file_subfolder)

                    elif filename.endswith("tar.gz") or filename.endswith("tgz") or filename.endswith("tar"):
                        tar = tarfile.open(fn,"r:gz")
                        tar.extractall(file_subfolder)
                        tar.close()

                except Exception as e:
                    return jsonify(e)
                
                os.remove(save_path)
                
                res = orchestrator.boot_existing_urbit(patp)
                if res == 0:
                    return jsonify(200)
                else:
                    return jsonify(400)

                return jsonify(200)

        else:
            return jsonify(501)

    return jsonify(404)

# Login
@app.route("/login", methods=['POST'])
def login():
    res = orchestrator.handle_login_request(request.get_json())
    if res == 200:
        res = make_response(jsonify(res))
        res.set_cookie('sessionid', orchestrator.make_cookie())
    else:
        res = make_response(jsonify(res))

    return res

if __name__ == '__main__':
    debug_mode = False
    app.run(host='0.0.0.0', port=27016, threaded=True, debug=debug_mode, use_reloader=debug_mode)
