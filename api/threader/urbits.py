#from datetime import datetime
import socket
from time import sleep
from threading import Thread

from log import Log
#from utils import Utils

class UrbitsLoop:
    def __init__(self,state):
        self.count = 0
        self.state = state
        self.broadcaster = self.state['broadcaster']
        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

        self.urb = None
        while self.urb == None:
            try:
                self.urb = self.state['dockers']['urbit']
            except:
                sleep(0.5)

        for patp in self.config['piers'].copy():
            self.broadcaster.urbit_broadcast(patp, 'minio', 'link')
            self.broadcaster.urbit_broadcast(patp, 'minio', 'unlink')

            self.broadcaster.urbit_broadcast(patp, 'container','rebuild')
            self.broadcaster.urbit_broadcast(patp, 'container', 'status', "loading")
            self.broadcaster.urbit_broadcast(patp, 'meld', 'urth')
            self.broadcaster.urbit_broadcast(patp, 'click', 'exist', False)
            self.broadcaster.urbit_broadcast(patp, 'vere', 'version')

            self.broadcaster.urbit_broadcast(patp, 'startram', 'access', 'unregistered') # remote, local, to-remote, to-local
            self.broadcaster.urbit_broadcast(patp, 'startram', 'minio', 'unregistered') # registered, registering
            self.broadcaster.urbit_broadcast(patp, 'startram', 'urbit', 'unregistered') # registered, registering

    def run(self):
        for patp in self.config['piers'].copy():
            Thread(target=self._vere_version, args=(patp,), daemon=True).start()
            Thread(target=self._container, args=(patp,), daemon=True).start()
            Thread(target=self._url, args=(patp,), daemon=True).start()
            self.count += 1

    def _container(self, patp):
        # running  -  Urbit container is running
        # stopped  -  Urbit container is stopped
        # loading  -  Still waiting for information
        # booting  -  +code not ready
        status = "stopped"
        try:
            if self.urb.urb_docker.is_running(patp):
                if len(self.get_code(patp)) == 27:
                    status = "running"
                else:
                    status = "booting"
        except:
            pass
        self.broadcaster.urbit_broadcast(patp, 'container', 'status', status)

    def _url(self, patp):
        try:
            cfg = self.urb._urbits[patp]
            url = f'http://{socket.gethostname()}.local:{cfg["http_port"]}'
            if cfg['network'] == 'wireguard':
                url = f"https://{cfg['wg_url']}"
        except Exception as e:
            url = ""
        self.broadcaster.urbit_broadcast(patp, 'container', 'url', url)

    def _vere_version(self, patp):
        if self.count == 0 or self.count % 30 == 0:
            try:
                if self.urb.urb_docker.is_running(patp):
                    res = self.urb.urb_docker.exec(patp, 'urbit --version')
                    if res:
                        res = res.output.decode("utf-8").strip().split("\n")[0]
                        self.broadcaster.urbit_broadcast(patp, 'vere', 'version', str(res))
            except Exception as e:
                self.broadcaster.urbit_broadcast(patp, 'vere', 'version', f'error: {e}')
