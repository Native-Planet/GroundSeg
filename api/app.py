import threading, time
from datetime import datetime
from flask import Flask, jsonify, request 
from flask_cors import CORS
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
app.config['TEMP_FOLDER'] = './tmp/'

# Todo: Look into what this actually does
CORS(app)

def meld_loop():
    while True:
        for p in orchestrator._urbits.values():
            now = int(datetime.utcnow().timestamp())

            if p.config['meld_schedule']:
                if int(p.config['meld_next']) < now:
                    x = orchestrator.get_urbit_loopback_addr(p.config['pier_name'])
                    p.send_meld(x)

threading.Thread(target=meld_loop).start()

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


if __name__ == '__main__':
    debug_mode = False
    app.run(host='0.0.0.0', port=27016, debug=True, use_reloader=debug_mode)
