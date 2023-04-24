from log import Log

# Action imports
from action_meld_urth import MeldUrth
from action_minio_link import MinIOLink

class WSUrbits:
    def __init__(self, config, structure, urb):
        self.config_object = config
        self.config = config.config
        self.structure = structure
        self.urb = urb
        self._urbits = self.urb._urbits 

        for patp in self.config['piers']:
            self.set_action(patp, 'meld', 'urth')
            self.set_action(patp, 'minio', 'link')
            self.set_action(patp, 'minio', 'unlink')
        Log.log("WS: Data ready for broadcast")

    # send to structure dict
    def set_action(self, patp, module, action, info=""):
        # Set patp to dict
        try:
            not_exist = (patp not in self.structure['urbits'])
            not_dict = (not isinstance(self.structure['urbits'], dict))
            if not_exist or not_dict:
                self.structure['urbits'][patp] = {}
        except Exception as e:
            Log.log(f"WS: ship '{patp}' failed to be added to broadcast dump: {e}")
            return False

        # Set module to dict
        try:
            not_exist = (module not in self.structure['urbits'][patp])
            not_dict = (not isinstance(self.structure['urbits'][patp], dict))
            if not_exist or not_dict:
                self.structure['urbits'][patp][module] = {}
        except Exception as e:
            Log.log(f"WS: module '{patp}:{module}' failed to be added to broadcast dump: {e}")
            return False
        # Set action to current value
        try:
            self.structure['urbits'][patp][module][action] = str(info)
        except Exception as e:
            Log.log(f"WS: action '{patp}:{module}:{action} {info}' failed to be added to broadcast dump: {e}")
            return False
        return True

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
        self.set_action(patp, 'meld', 'urth','initializing')
        MeldUrth(self, patp, self.urb).run()

    # TODO: remove unlink stuff
    def minio_link(self, pier_config, acc="", secret="", bucket="", unlink=False):
        MinIOLink(self, self.urb, unlink).link(pier_config, acc, secret, bucket)
