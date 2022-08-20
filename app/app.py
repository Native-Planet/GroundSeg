import copy, json
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response
from flask import render_template, make_response
from werkzeug.utils import secure_filename

from orchestrator import Orchestrator
import urbit_docker
#import urbit_options


def signal_handler(sig, frame):
    print("Exiting gracefully")
    cmds = shlex.split("./kill_urbit.sh")
    print(cmds)
    p = subprocess.Popen(cmds,shell=True)
    sys.exit(0)


orchestrator = Orchestrator("settings/system.json")


app = Flask(__name__)
#app.register_blueprint(system_setup.app)
#app.register_blueprint(urbit_options.app)

@app.route("/")
def main():
    running, stopped = orchestrator.getUrbits()
    return render_template('urbit.html', running_piers = running, stopped_piers=stopped)

@app.route('/urbit/access', methods=['GET'])
def pier_access():
    pier = request.args.get('pier')
    urbit = orchestrator._urbits[pier]

    return redirect("http://192.168.0.229:8080") # TODO dont hardcode this

@app.route('/urbit/pier', methods=['GET'])
def pier_info():
    pier = request.args.get('pier')
    urbit = orchestrator._urbits[pier]
    code = urbit.get_code().decode('utf-8')

    return render_template('pier.html', name=pier, code = code, running = urbit.isRunning())

@app.route('/upload/key',methods=['GET','POST'])
def uploadKey():
    if request.method == 'GET':
        return render_template('upload_key.html')
    if request.method == 'POST':
        patp = request.form['patp']
        key = request.form['key']

        data = copy.deepcopy(urbit_docker.default_pier_config)
        data['pier_name'] = patp
        with open(f'settings/{patp}.json', 'w') as f:
            json.dump(data, f, indent = 4)
        
        urbit = urbit_docker.UrbitDocker(data)
        urbit.addKey(key)
        orchestrator.addUrbit(patp, urbit)
        return redirect("/")


@app.route("/urbit/start", methods=['POST'])
def start_pier():
    if request.method == 'POST':
        pier = request.form['boot']
        urbit = orchestrator._urbits[pier]
        if(urbit==None):
            return Response("Pier not found", status=400)
        urbit.start()
    return redirect("/")

@app.route("/urbit/stop", methods=['POST'])
def stop_pier():
    if request.method == 'POST':
        for p in request.form:
            urbit = orchestrator._urbits[p]
            if(urbit==None):
                return Response("Pier not found", status=400)
            urbit.stop()
        
    return redirect("/")


@app.route("/main")
def mainscreen():

    return render_template('urbit.html')

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0')
