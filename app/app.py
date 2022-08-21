import copy, json
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response
from flask import render_template, make_response
from werkzeug.utils import secure_filename

from orchestrator import Orchestrator
import urbit_docker
import upload_api, urbit_api, settings_api


def signal_handler(sig, frame):
    print("Exiting gracefully")
    cmds = shlex.split("./kill_urbit.sh")
    print(cmds)
    p = subprocess.Popen(cmds,shell=True)
    sys.exit(0)


orchestrator = Orchestrator("settings/system.json")


app = Flask(__name__)

app.config['ORCHESTRATOR'] = orchestrator
app.config['TEMP_FOLDER'] = './tmp/'


app.register_blueprint(urbit_api.app)
app.register_blueprint(upload_api.app)
app.register_blueprint(settings_api.app)

@app.route("/", methods=['POST','GET'])
def mainscreen():
    piers = orchestrator.getUrbits()
    return render_template('urbit.html', piers = piers)


if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0')
