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
#app.config['ORCHESTRATOR'] = orchestrator
app.config['TEMP_FOLDER'] = './tmp/'

# Todo: Look into what this actually does
CORS(app)


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


if __name__ == '__main__':
    debug_mode = False
    app.run(host='0.0.0.0', port=27016, debug=debug_mode, use_reloader=debug_mode)
