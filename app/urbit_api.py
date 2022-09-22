import requests, copy, json, shutil
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response, Blueprint
from flask import render_template, make_response, send_file, jsonify
from flask import current_app


import os,time
import zipfile, tarfile
import glob
from io import BytesIO
import zipfile
from werkzeug.utils import secure_filename
import socket

from orchestrator import Orchestrator

import urbit_docker


app = Blueprint('urbit', __name__, template_folder='templates')

@app.route('/urbit/pier', methods=['GET'])
def pier_info():
    pier = request.args.get('pier')
    orchestrator = current_app.config['ORCHESTRATOR']
    urbits = orchestrator.getUrbits()
    
    urbit=None
    for u in urbits:
        if(pier == u['name']):
            urbit = u

    if(urbit == None):
        return jsonify(400)

    nw_label = "Local"
    if(urbit['network'] == 'wireguard'):
        nw_label = "Remote"

    p = dict()
    p['nw_label'] = nw_label
    p['pier'] = urbit
    p['wg_reg'] = orchestrator.wireguard_reg
    p['wg_running'] = orchestrator.wireguard.isRunning()

    return(jsonify(p))

@app.route('/urbit/code', methods=['GET'])
def pier_code():
    pier = request.args.get('pier')
    orchestrator = current_app.config['ORCHESTRATOR']
    code = orchestrator.getCode(pier)
    return jsonify(code)

@app.route('/urbit/minio/register', methods=['POST'])
def register_minio():
    patp = request.form['patp']
    password = request.form['password']

    orchestrator = current_app.config['ORCHESTRATOR']
    x = orchestrator.registerMinIO(patp, password)

    if x == 0:
        return jsonify(200)

    return jsonify(400)

@app.route("/urbit/network", methods=['POST'])
def set_network():
    for pier in request.form:
        current_app.config['ORCHESTRATOR'].switchUrbitNetwork(pier)
        return jsonify(200)

@app.route("/urbit/start", methods=['POST'])
def start_pier():
    url = "/"
    print(request.form)
    for p in request.form:
        urbit = current_app.config['ORCHESTRATOR']._urbits[p]
        if(urbit==None):
            return Response("Pier not found", status=400)
        urbit.start()
        url = f'/urbit/pier?pier={urbit.pier_name}'
        time.sleep(2)
            
    # pier started
    return jsonify(200)

@app.route("/urbit/stop", methods=['POST'])
def stop_pier():
    url = "/"
    if request.method == 'POST':
        print(request.form)
        for p in request.form:
            urbit = current_app.config['ORCHESTRATOR']._urbits[p]
            if(urbit==None):
                return Response("Pier not found", status=400)
            urbit.stop()
            url = f'/urbit/pier?pier={urbit.pier_name}'
            
    # pier stopped
    return jsonify(200)

@app.route("/urbit/minio_endpoint", methods=['POST'])
def set_minio_endpoint():
    ship = request.form['pier']

    orchestrator = current_app.config['ORCHESTRATOR']
    orchestrator.setMinIOEndpoint(ship)

    return jsonify(200)

@app.route("/urbit/minio_secret", methods=['POST'])
def get_minio_secret():
    patp = request.form['patp']
    orchestrator = current_app.config['ORCHESTRATOR']
    x = orchestrator.getMinIOSecret(patp)
    print(x)

    return jsonify(x)


@app.route("/urbit/eject", methods=['POST'])
def eject_pier():
    for p in request.form:
        urbit = current_app.config['ORCHESTRATOR']._urbits[p]
        if(urbit==None):
            return Response("Pier not found", status=400)
        if(urbit.isRunning()):
            print('stopping urbit')
            urbit.stop()
        fileName = f'{urbit.pier_name}.zip'
        memory_file = BytesIO()
        file_path=f'{urbit._volume_directory}/{urbit.pier_name}/_data/'
        print('compressing')
        with zipfile.ZipFile(memory_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(file_path):
                arc_dir = root[root.find('_data/')+6:]
                for file in files:
                    zipf.write(os.path.join(root, file), arcname=os.path.join(arc_dir,file))
        memory_file.seek(0)
        return send_file(memory_file,
                 download_name=fileName,
                 as_attachment=True)

    return jsonify(400)


@app.route("/urbit/delete", methods=['POST'])
def delete_pier():
    for p in request.form:
        current_app.config['ORCHESTRATOR'].removeUrbit(p)
    return jsonify(200)
