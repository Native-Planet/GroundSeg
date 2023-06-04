from time import sleep
from threading import Thread

from log import Log

# Action imports
#from action_meld_urth import MeldUrth
#from action_minio_link import MinIOLink
#from action_access_toggle import AccessToggle

class WSUrbits:
    def __init__(self, state): 
        self.state = state
        self.broadcaster = self.state['broadcaster']

        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

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
    #   interactions with the Urbit Docker container
    #

    def start(self, patp, act):
        ship = self._urbits[patp]
        arch = self.config_object._arch
        vol = self.urb._volume_directory
        key = ''
        res = self.urb.urb_docker.start(ship, arch, vol, key, act)
        return res

    def remove_container(self, patp):
        return self.urb.urb_docker.remove_container(patp)

    def create_container(self, patp):
        ship = self._urbits[patp]
        image = self.temp_image(patp)
        vol = self.urb._volume_directory
        key = ''
        res = self.urb.urb_docker.create(ship, image, vol, key)
        return res

    def temp_image(self, patp):
        repo = self._urbits[patp]['urbit_repo']
        tag = self._urbits[patp]['urbit_version']
        image = f"{repo}:{tag}"

        arch = self.config_object._arch
        sha = f"urbit_{arch}_sha256"
        hash_str = self._urbits[patp][sha]

        if hash_str != "" and hash_str is not None:
            image = f"{image}@sha256:{hash_str}"

        return image


    # TODO: Make this its own action file

    #
    #   Actions
    #

    # TODO: Make this its own action file
    def container_rebuild(self, patp):
        # removing      -  deleting old container
        # starting      -  starting ship
        # success       -  ship has started
        # failure\n<err> -  Failure message
        success = False
        try:
            self.ws_util.urbit_broadcast(patp, 'container', 'rebuild', 'removing')
            running = self.urb.urb_docker.is_running(patp)
            if self.remove_container(patp):
                self.ws_util.urbit_broadcast(patp, 'container', 'rebuild', 'rebuilding')
                if self.create_container(patp):
                    success = True
                    if running:
                        self.ws_util.urbit_broadcast(patp, 'container', 'rebuild', 'starting')
                        res = self.start(patp,'boot')
                        success = res == "succeeded"
            if success:
                self.ws_util.urbit_broadcast(patp, 'container', 'rebuild', 'success')
        except Exception as e:
            Log.log(f"{patp}:container:rebuild Failed to rebuild container {e}")
            self.ws_util.urbit_broadcast(patp, 'container', 'rebuild', f'failure\n{e}')

        import time
        time.sleep(3)
        self.ws_util.urbit_broadcast(patp, 'container', 'rebuild')

    def meld_urth(self, patp):
        self.ws_util.urbit_broadcast(patp, 'meld', 'urth','initializing')
        MeldUrth(self, patp, self.urb, self.ws_util).run()

    # TODO: remove unlink stuff
    def minio_link(self, pier_config, acc="", secret="", bucket="", unlink=False):
        MinIOLink(self.urb, self.ws_util, unlink).link(pier_config, acc, secret, bucket)

    def access_toggle(self, patp, t=None):
        if t:
            AccessToggle(patp, self.config_object, self.urb, self.ws_util).set(t)
        else:
            AccessToggle(patp, self.config_object, self.urb, self.ws_util).toggle()
