import requests, copy, json, shutil
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response, Blueprint
from flask import render_template, make_response, jsonify
from flask import current_app

import os
import zipfile, tarfile
import glob
import psutil
from werkzeug.utils import secure_filename

from orchestrator import Orchestrator

import urbit_docker

#import system_info as sys_info


app = Blueprint('settings', __name__, template_folder='templates')

@app.route('/settings',methods=['GET'])
def settings():
    orchestrator = current_app.config['ORCHESTRATOR']
    ram = psutil.virtual_memory()
    cpu = psutil.cpu_percent(interval=0.1)
    temp = psutil.sensors_temperatures()['coretemp'][0].current
    disk = shutil.disk_usage("/")

    return jsonify({
        "ram": ram.percent,
        "disk" : disk,
        "temp" : temp,
        "anchor" : orchestrator.wireguard.isRunning(),
        # TODO
        "ethOnly" : False,
        "connected" : "Native Planet 5G", 
        "networks" : ["John's Wifi","City Wok","Native Planet 5G"],
        "minio" : True # true if online
    })
    
@app.route('/settings/anchor',methods=['POST'])
def anchor_status():
    orchestrator = current_app.config['ORCHESTRATOR']
    isOn = request.form['anchor']
    # isOn gets sent as a string
    if isOn == 'true':
        orchestrator.wireguard.start()
    else:
        orchestrator.wireguard.stop()

    return jsonify(200)

# register anchor key
@app.route('/settings/anchor/register',methods=['POST'])
def anchor_register():
    key = request.form['key']
    # TODO
    # return jsonify(400) # needed for fail?
    return jsonify(200)

# toggle ethernet only
@app.route('/settings/eth-only',methods=['POST'])
def ethernet_only():
    isEthOnly = request.form['ethernet']
    if isEthOnly == 'true':
        print('set to ethernet only')
        # toggle ethernet only
    else:
        print('set to wifi and ethernet')
        # toggle wifi and ethernet

    return jsonify(200)

# connect to wifi network
@app.route('/settings/connect',methods=['POST'])
def connect_wifi():
    network = request.form['network']
    password = request.form['password']
    # connect to network
    
    return jsonify(200)

# restart minIO
@app.route('/settings/minio',methods=['POST'])
def restart_minio():
    # restart minio
    return jsonify(200)

@app.route('/settings/logs',methods=['POST','GET'])
def settings_logs():
    orchestrator = current_app.config['ORCHESTRATOR']
    container_logs = orchestrator.getContainers()
    if request.method == 'GET':
        return jsonify(container_logs)
    if request.method == 'POST':
        container = request.form['logs']
        print(container)
        log = orchestrator.getLogs(container).decode('utf-8')

        return jsonify(log)

@app.route('/settings/shutdown',methods=['POST'])
def shutdown():
    os.system('shutdown now')
    return jsonify(200)

@app.route('/settings/restart',methods=['POST'])
def restart():
    os.system('shutdown -r -h `date --date "now + 30 seconds"`')
    return jsonify(200)
