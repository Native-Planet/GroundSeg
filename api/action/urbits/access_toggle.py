from log import Log

class AccessToggle:
    def __init__(self, patp, state):
        #config, urb, ws_util):
        self.patp = patp
        '''
        self.config_object = config
        self.config = config.config
        self.urb = urb
        self.ws_util = ws_util
        '''

    '''
    def broadcast(self, t):
        self.ws_util.urbit_broadcast(self.patp,'startram', 'access', t)

    def log(self, t):
        Log.log(f"ws_urbits:access_toggle:{self.patp} {t}")
    '''

    def toggle(self):
        if self.urb._urbits[self.patp]['network'] == "wireguard":
            self.set("local")
        else:
            self.set("remote")

    def set(self, t):
        print(t)
        print(t)
        print(t)
        print(t)
        print(t)
        print(t)
        print(t)
        '''
        patp = self.patp
        wg_reg = self.config['wgRegistered']
        wg_is_running = self.urb.wg.is_running()
        c = self.urb.urb_docker.get_container(patp)
        if c:
            running = False
            if c.status == "running":
                running = True

            self.urb.urb_docker.remove_container(patp)
            if t == "remote":
                self.urb._urbits[patp]['network'] = "wireguard"
                self.broadcast("to-remote")
            else:
                self.urb._urbits[patp]['network'] = "none"
                self.broadcast("to-local")

            self.log(f"Network set to {self.urb._urbits[patp]['network']}")
            self.urb.save_config(patp)

            created = self.urb.urb_docker.start(self.urb._urbits[patp],
                                                self.config_object._arch,
                                                self.urb._volume_directory
                                                )
            if (created == "succeeded") and running:
                self.urb.start(patp)

            self.broadcast(t)
        '''
