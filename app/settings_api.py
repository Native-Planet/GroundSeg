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
#    container_logs = orchestrator.getContainers()
    ram = psutil.virtual_memory()
    cpu = psutil.cpu_percent(interval=0.1)
    temp = psutil.sensors_temperatures()['coretemp'][0].current
    disk = shutil.disk_usage("/")

    return jsonify({
        "ram": ram.percent,
        "disk" : disk,
        "temp" : temp,
        "anchor" : orchestrator.wireguard.isRunning()
    })
    
@app.route('/settings/anchor',methods=['POST'])
def anchor_status():
    orchestrator = current_app.config['ORCHESTRATOR']
    isOn = request.form['anchor']
    if(isOn):
        orchestrator.wireguard.start()
    else:
        orchestrator.wireguard.stop()

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
