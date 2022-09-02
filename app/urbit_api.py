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

from orchestrator import Orchestrator

import urbit_docker


app = Blueprint('urbit', __name__, template_folder='templates')


@app.route('/urbit/access', methods=['GET'])
def pier_access():
    pier = request.args.get('pier')
    urbit = current_app.config['ORCHESTRATOR']._urbits[pier]

    url = f"http://nativeplanet.local:{urbit.config['http_port']}"
    
    if(urbit.config['network']=='wireguard'):
        url = f"http://{urbit.config['wg_url']}"
    
    return redirect(url)

@app.route('/urbit/pier', methods=['GET'])
def pier_info():
    pier = request.args.get('pier')
    urbits = current_app.config['ORCHESTRATOR'].getUrbits()
    
    urbit=None
    for u in urbits:
        if(pier == u['name']):
            urbit = u

    if(urbit == None):
        return Response("Pier not found", status=400)

    nw_label = "Local"
    if(u['network'] == 'wireguard'):
        nw_label = "Remote"

    p = dict()
    p['nw_label'] = nw_label
    p['pier'] = urbit

    return(jsonify(p))


@app.route("/urbit/network", methods=['POST'])
def set_network():
    for pier in request.form:
        current_app.config['ORCHESTRATOR'].switchUrbitNetwork(pier)
        return jsonify(200)

@app.route("/urbit/start", methods=['POST'])
def start_pier():
    url = "/"
    if request.method == 'POST':
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

@app.route("/urbit/eject", methods=['POST'])
def eject_pier():
    if request.method == 'POST':
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
    return Response("Pier not found", status=400)



@app.route("/urbit/delete", methods=['POST'])
def delete_pier():
    if request.method == 'POST':
        for p in request.form:
            current_app.config['ORCHESTRATOR'].removeUrbit(p)
             
    # pier deleted
    return jsonify(200)




