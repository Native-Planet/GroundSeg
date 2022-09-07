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
from wifi import Cell, Scheme
from pprint import pprint


app = Blueprint('settings', __name__, template_folder='templates')

glob_network=[]

@app.route('/settings',methods=['GET'])
def settings():
    orchestrator = current_app.config['ORCHESTRATOR']
    ram = psutil.virtual_memory()
    cpu = psutil.cpu_percent(interval=0.1)
    temp = psutil.sensors_temperatures()['coretemp'][0].current
    disk = shutil.disk_usage("/")
    eth = True
    net = psutil.net_if_stats()

    for k,v in net.items():
        if 'wl' in k:
            wifi = k
            if(v.isup):
                eth = False
                break
            

    return jsonify({
        "ram": ram.percent,
        "disk" : disk,
        "temp" : temp,
        "anchor" : orchestrator.wireguard.isRunning(),
        # TODO
        "eth-only" : eth,
        "connected" : "Native Planet 5G", 
        "minio" : orchestrator.minIO_on
    })
    
@app.route('/settings/networks', methods=['GET'])
def list_networks():
    wifi = 'wl'
    net = psutil.net_if_stats()

    for k,v in net.items():
        if 'wl' in k:
            wifi = k

    global glob_network
    networks = []

    print(list(Cell.all(wifi)))

    try:
        n = list(Cell.all(wifi))
        for c in n:
            if((c.ssid !='') and ('\\x00\\x00' not in c.ssid )):
                if(c.ssid not in networks):
                    networks.append(c.ssid)
        glob_network = networks
    except Exception as e:
        networks = glob_network
        pass

    print(networks)
    return jsonify(networks)

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
    orchestrator = current_app.config['ORCHESTRATOR']
    orchestrator.registerDevice(key)
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
    orchestrator = current_app.config['ORCHESTRATOR']
    orchestrator.stopMinIOs()
    orchestrator.startMinIOs()

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
