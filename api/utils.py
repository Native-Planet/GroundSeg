# Python
import ssl
import urllib.request
import requests
from time import sleep

# GroundSeg modules
from log import Log
from binary_updater import BinUpdater

#import sys
#import os
#import nmcli
#import docker

#from datetime import datetime

class Utils:
    def check_internet_access():
        try:
            context = ssl._create_unverified_context()
            urllib.request.urlopen('https://nativeplanet.io',
                                   timeout=1,
                                   context=context)

            return True

        except Exception as e:
            Log.log("Check internet access error: {e}")
            return False

    def get_version_info(config, debug_mode):
        Log.log("Updater thread started")
        while True:
            try:
                Log.log("Checking for updates")
                url = config.config['updateUrl']
                r = requests.get(url)

                if r.status_code == 200:
                    config.update_avail = True
                    config.update_payload = r.json()

                    # Run binary updater
                    b = BinUpdater()
                    b.check_bin_update(config, debug_mode)

                    if config.gs_ready:
                        print("docker update here")
                        # Run docker updates
                        sleep(config.config['updateInterval'])
                    else:
                        sleep(60)

                else:
                    raise ValueError(f"Status code {r.status_code}")

            except Exception as e:
                config.update_avail = False
                Log.log(f"Unable to retrieve update information: {e}")
                sleep(60)

    '''
    def remove_urbit_containers():
        client = docker.from_env()

        # Force remove containers
        containers = client.containers.list(all=True)
        for container in containers:
            try:
                if container.image.tags[0] == "tloncorp/urbit:latest":
                    container.remove(force=True)
                if container.image.tags[0] == "tloncorp/vere:latest":
                    container.remove(force=True)
            except:
                pass

        # Check if all have been removed
        containers = client.containers.list(all=True)
        count = 0
        for container in containers:
            try:
                if container.image.tags[0] == "tloncorp/urbit:latest":
                    count = count + 1
                if container.image.tags[0] == "tloncorp/vere:latest":
                    count = count + 1
            except:
                pass
        return count == 0


class Network:

    def list_wifi_ssids():
        return [x.ssid for x in nmcli.device.wifi() if len(x.ssid) > 0]

    def wifi_connect(ssid, pwd):
        try:
            nmcli.device.wifi_connect(ssid, pwd)
            return "success"
        except Exception as e:
            return f"failed: {e}"
'''
