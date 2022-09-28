import requests, copy, json, shutil, time
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response, Blueprint
from flask import render_template, make_response, jsonify
from flask import current_app

import os, subprocess
import zipfile, tarfile
import glob
import psutil
from werkzeug.utils import secure_filename

from orchestrator import Orchestrator

import urbit_docker
from wifi import Cell, Scheme
from wifi_finder import Finder
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
    net = psutil.net_if_stats()

    check_connected = subprocess.Popen(['iwgetid','-r'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
    connected, stderr = check_connected.communicate()

    wifi_status = subprocess.Popen(['nmcli','radio','wifi'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
    ws, stderr = wifi_status.communicate()

    if ws == b'enabled\n':
        eth_only = False
    else:
        eth_only = True

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
        "ethOnly" : eth_only,
        "connected" : connected.decode("utf-8"),
        "minio" : orchestrator.minIO_on,
        "wg_reg" : orchestrator.wireguard_reg,
        "gsVersion" : orchestrator.config['gsVersion']
    })
    
@app.route('/settings/update', methods=['GET','POST'])
def has_update():
    orchestrator = current_app.config['ORCHESTRATOR']

    if request.method == 'GET':
        x = orchestrator.checkForUpdate()
        return jsonify(x)

    if request.method == 'POST':
        orchestrator.downloadUpdate()
        return jsonify(200)

@app.route('/settings/networks', methods=['GET'])
def list_networks():
    wifi = 'wl'
    net = psutil.net_if_stats()

    for k,v in net.items():
        if 'wl' in k:
            wifi = k

    global glob_network
    networks = []

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

    return jsonify(networks)

@app.route('/settings/anchor',methods=['POST'])
def anchor_status():
    orchestrator = current_app.config['ORCHESTRATOR']
    isOn = request.form['anchor']
    # isOn gets sent as a string
    if isOn == 'true':
        orchestrator.wireguardStart()
    else:
        orchestrator.wireguardStop()

    return jsonify(200)

# register anchor key
@app.route('/settings/anchor/register',methods=['POST'])
def anchor_register():
    key = request.form['key']
    orchestrator = current_app.config['ORCHESTRATOR']
    out =  orchestrator.registerDevice(key)
    if out == 0:
        return jsonify(200)
    else:
        return jsonify(400)

# change anchor endpoint
@app.route('/settings/anchor/endpoint',methods=['GET','POST'])
def anchor_endpoint():
    orchestrator = current_app.config['ORCHESTRATOR']

    if request.method == 'GET':
        current_endpoint = orchestrator.getWireguardUrl() 
        return jsonify(current_endpoint)

    if request.method == 'POST':
        endpoint = request.form['new']
        x = orchestrator.changeWireguardUrl(endpoint)
        if x == 0:
            return jsonify(200)
    return jsonify(400)

# toggle ethernet only
@app.route('/settings/eth-only',methods=['POST'])
def ethernet_only():
    isEthOnly = request.form['ethernet']
    if isEthOnly == 'true':
        os.system('nmcli radio wifi off')
        print('set to ethernet only')
    else:
        os.system('nmcli radio wifi on')
        print('set to wifi and ethernet')

    return jsonify(200)

# connect to wifi network
@app.route('/settings/connect',methods=['POST'])
def connect_wifi():
    network = request.form['network']
    password = request.form['password']
    connected = request.form['connected']

    connect_attempt = subprocess.Popen(['nmcli','dev','wifi','connect',network,'password',password],
            stdout=subprocess.PIPE,stderr=subprocess.STDOUT)

    did_connect, stderr = connect_attempt.communicate()
    did_connect = did_connect.decode("utf-8")[0:5]

    if did_connect == 'Error':
        print('400')
        return jsonify(400)
    else:
        # for some reason deleting doesn't work
        #if connected != '':
            #del_connection = subprocess.Popen(['nmcli','con','del',connected],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)

            #did_delete, delerr = del_connection.communicate()
            #did_delete = did_delete.decode("utf-8")
            #print(did_delete)

        return jsonify(200)

# restart minIO
@app.route('/settings/minio',methods=['POST'])
def restart_minio():
    orchestrator = current_app.config['ORCHESTRATOR']
    orchestrator.stopMinIOs()
    orchestrator.startMinIOs()

    time.sleep(1)
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
    os.system('reboot')
    return jsonify(200)
