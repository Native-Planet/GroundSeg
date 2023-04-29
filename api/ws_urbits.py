from log import Log

# Action imports
from action_meld_urth import MeldUrth
from action_minio_link import MinIOLink

class WSUrbits:
    def __init__(self, config, urb, ws_util):
        self.config_object = config
        self.config = config.config
        self.structure = ws_util.structure
        self.urb = urb
        self._urbits = self.urb._urbits 
        self.ws_util = ws_util

        for patp in self.config['piers']:
            self.ws_util.urbit_broadcast(patp, 'meld', 'urth')
            self.ws_util.urbit_broadcast(patp, 'minio', 'link')
            self.ws_util.urbit_broadcast(patp, 'minio', 'unlink')

    #
    #   interacting with self._urbits dict (config)
    #

    def get_config(self, patp, key):
        try:
            return self._urbits[patp][key]
        except:
            return None

    def set_config(self, patp, key, value):
        try:
            old_value = self._urbits[patp][key]
            self._urbits[patp][key] = value
            Log.log(f"WS: {patp}: '{key}':{old_value} -> '{key}':{value}")
            self.urb.save_config(patp)
            return True
        except Exception as e:
            Log.log(f"WS: {patp} set config failed: {e}")
        return False

    #
    #   actions sent to the Urbit container
    #

    def start(self, patp, act):
        ship = self._urbits[patp]
        arch = self.config_object._arch
        vol = self.urb._volume_directory
        key = ''
        res = self.urb.urb_docker.start(ship, arch, vol, key, act)
        return res

    def meld_urth(self, patp):
        self.ws_util.urbit_broadcast(patp, 'meld', 'urth','initializing')
        MeldUrth(self, patp, self.urb, self.ws_util).run()

    # TODO: remove unlink stuff
    def minio_link(self, pier_config, acc="", secret="", bucket="", unlink=False):
        MinIOLink(self.urb, self.ws_util, unlink).link(pier_config, acc, secret, bucket)
