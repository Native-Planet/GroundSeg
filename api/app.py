import copy, json
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response
from flask import render_template, make_response, jsonify
from flask_cors import CORS
from werkzeug.utils import secure_filename
from graphene import Schema

from orchestrator import Orchestrator
from schema import Query

def signal_handler(sig, frame):
    print("Exiting gracefully")
    cmds = shlex.split("./kill_urbit.sh")
    print(cmds)
    p = subprocess.Popen(cmds,shell=True)
    sys.exit(0)

orchestrator = Orchestrator("/settings/system.json")

app = Flask(__name__)
CORS(app)

app.config['ORCHESTRATOR'] = orchestrator
app.config['TEMP_FOLDER'] = './tmp/'

@app.route("/graphql", methods=['POST'])
def graphql_endpoint():
    schema = Schema(query=Query)

    query_string = request.get_json()
    result = schema.execute(query_string['data'])
    return jsonify(result.data)

if __name__ == '__main__':
    debug_mode = False
    app.run(host='0.0.0.0', port=27016, debug=debug_mode, use_reloader=debug_mode)
