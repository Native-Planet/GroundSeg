from wireguard_docker import WireguardDocker
from urbit_docker import UrbitDocker

import requests
import subprocess
import base64

import urbit_docker
import wireguard_docker
import minio_docker

import sys
import time
import json

## systme setup
patp='walzod-fogsed-mopfel-winrux'

## Setup Wireguard pubkey
#subprocess.run("wg genkey > privkey", shell=True)
#subprocess.run("cat privkey| wg pubkey | base64 -w 0 > pubkey", shell=True)

"""
with open('pubkey') as f:
    pubkey = f.read().strip()
with open('privkey') as f:
    privkey = f.read().strip()

## call retrieve api

url = "https://api1.nativeplanet.live/v1"


headers = {"Content-Type": "application/json"}

update_data = {
    "patp" : f"{patp}",
    "pubkey":pubkey
}

### Check and see if pubkey exists
response = None
try:
    response = requests.get(f'{url}/retrieve?pubkey={pubkey}', headers=headers).json()
    print(response)
except Exception as e:
    print(e)


# If does not exist
if response['error']==1 or response['status'] != 'ready':
    # create it
    try:
        response = requests.post(f'{url}/create', json = update_data, headers=headers).json()
        print(response)
    except Exception as e:
        print(e)

#    sys.exit()

    # wait for it to be created
    while response['status'] != 'ready':
        try:
            response = requests.get(f'{url}/retrieve?pubkey={pubkey}', headers=headers).json()
            print(response)
        except Exception as e:
            print(e)
        print("Waiting for endpoint to be created")
        if(response['status'] != 'ready'):
            time.sleep(60)


# get and decode configuration
config = base64.b64decode(response['conf']).decode('utf-8')

config = config.replace('privkey', privkey)

"""

## Start WG docker
filename = "settings/wireguard.json"
f = open(filename)
data = json.load(f)
wg = WireguardDocker(data)
wg.addConfig(config)
wg.start()


## Start Urbit
filename = "settings/walzod-fogsed-mopfel-winrux.json"
f = open(filename)
data = json.load(f)
urdock = UrbitDocker(data)
urdock.start()
