#import requests, json
import threading, time, os, zipfile, tarfile, copy, shutil
from datetime import datetime
from flask import Flask, jsonify, request 
from flask_cors import CORS
from werkzeug.utils import secure_filename
from orchestrator import Orchestrator

# Todo: is this even used?
def signal_handler(sig, frame):
    print("Exiting gracefully")
    cmds = shlex.split("./kill_urbit.sh")
    print(cmds)
    p = subprocess.Popen(cmds,shell=True)
    sys.exit(0)

# Load GroundSeg
orchestrator = Orchestrator("/settings/system.json")

app = Flask(__name__)
app.config['TEMP_FOLDER'] = '/tmp/'

# Todo: Look into what this actually does
CORS(app)

# Checks if a meld is due, runs meld
def meld_loop():
    while True:
        for p in orchestrator._urbits.values():
            now = int(datetime.utcnow().timestamp())

            if p.config['meld_schedule']:
                if int(p.config['meld_next']) < now:
                    x = orchestrator.get_urbit_loopback_addr(p.config['pier_name'])
                    p.send_meld(x)

threading.Thread(target=meld_loop).start()

#
#   Endpoints
#

# Get all urbits
@app.route("/urbits", methods=['GET'])
def all_urbits():
    urbs = orchestrator.get_urbits()
    return jsonify(urbs)

# Handle urbit ID related requests
@app.route('/urbit', methods=['GET','POST'])
def urbit_info():
    urbit_id = request.args.get('urbit_id')
    
    if request.method == 'GET':
        urb = orchestrator.get_urbit(urbit_id)
    
        return jsonify(urb)

    if request.method == 'POST':
        res = orchestrator.handle_urbit_post_request(urbit_id, request.get_json())
        return orchestrator.custom_jsonify(res)

# Handle device's system settings
@app.route("/system", methods=['GET','POST'])
def system_settings():
    if request.method == 'GET':
        settings = orchestrator.get_system_settings()
        return jsonify(settings)

    if request.method == 'POST':
        module = request.args.get('module')
        res = orchestrator.handle_module_post_request(module, request.get_json())
        return jsonify(res)

# Handle anchor registration related information
@app.route("/anchor", methods=['GET'])
def anchor_settings():
    if request.method == 'GET':
        settings = orchestrator.get_anchor_settings()
        return jsonify(settings)


# Pier upload
@app.route("/upload", methods=['POST'])
def pier_upload():
    
    # Uploaded pier
    file = request.files['file']
    filename = secure_filename(file.filename)
    fn = save_path = f'/tmp/{filename}'
    current_chunk = int(request.form['dzchunkindex'])
        
    if os.path.exists(save_path) and current_chunk == 0:
        # 400 and 500s will tell dropzone that an error occurred and show an error
        os.remove(os.path.join(app.config['TEMP_FOLDER'], filename))
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
                        zip_ref.extractall('/tmp/')

                elif filename.endswith("tar.gz") or filename.endswith("tgz") or filename.endswith("tar"):
                    tar = tarfile.open(fn,"r:gz")
                    tar.extractall(app.config['TEMP_FOLDER'])
                    tar.close()

            except Exception as e:
                return jsonify(e)
            
            os.remove(os.path.join(app.config['TEMP_FOLDER'], filename))
            
            patp = filename.split('.')[0]
            res = orchestrator.boot_existing_urbit(patp)
            if res == 0:
                return jsonify(200)
            else:
                return jsonify(400)

            return jsonify(200)

    else:
        return jsonify(501)


if __name__ == '__main__':
    debug_mode = False
    app.run(host='0.0.0.0', port=27016, debug=debug_mode, use_reloader=debug_mode)
