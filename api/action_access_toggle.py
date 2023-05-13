from log import Log

class AccessToggle:
    def __init__(self, patp, config, urb, ws_util):
        self.patp = patp
        self.config_object = config
        self.config = config.config
        self.urb = urb
        self.ws_util = ws_util

    def broadcast(self, t):
        self.ws_util.urbit_broadcast(self.patp,'startram', 'access', t)

    def log(self, t):
        Log.log(f"ws_urbits:access_toggle:{self.patp} {t}")

    def toggle(self):
        if self.urb._urbits[self.patp]['network'] == "wireguard":
            self.set("local")
        else:
            self.set("remote")

    def set(self, t):
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
        wg_reg = self.config['wgRegistered']
        wg_is_running = self.wg.is_running()
        c = self.urb_docker.get_container(patp)

        if c:
            try:
                running = False
                if c.status == "running":
                    running = True
                
                old_network = self._urbits[patp]['network']

                self.urb_docker.remove_container(patp)

                if old_network == "none" and wg_reg and wg_is_running:
                    self._urbits[patp]['network'] = "wireguard"
                else:
                    self._urbits[patp]['network'] = "none"

                Log.log(f"{patp}: Network changed: {old_network} -> {self._urbits[patp]['network']}")
                self.save_config(patp)

                created = self.urb_docker.start(self._urbits[patp],
                                                self.config_object._arch,
                                                self._volume_directory
                                                )
                if (created == "succeeded") and running:
                    self.start(patp)

                return 200

            except Exception as e:
                Log.log(f"{patp}: Unable to change network: {e}")
        '''
        return 400
