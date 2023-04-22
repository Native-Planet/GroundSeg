from time import sleep
from log import Log

class MeldUrth:
    def __init__(self, patp, urb, set_action):
        self.patp = patp
        self.urb = urb
        self.set_action = set_action
        # TODO: Avoid interacting with urb_docker directly
        self.urb_docker = urb.urb_docker

    def full(self, start):
        # start is container start command (see ws_urbits.py)
        patp = self.patp
        running = self.stop_running(patp)
        devmode = self.stop_devmode(patp)

        try:
            if self.action(patp,'pack', start):
                if self.action(patp,'meld', start):
                    self.revert_devmode(patp, devmode)
                    if self.revert_running(patp, running, start):
                        Log.log(f"{patp}: Urth Pack and Meld action completed")
                        self.broadcast("success")
                    else:
                        raise Exception()
                else:
                    raise Exception()
            else:
                raise Exception()
        except Exception:
            Log.log(f"{patp}: Urth Pack and Meld action did not complete")
            self.broadcast("failure")

        sleep(3)
        self.broadcast("")

    def broadcast(self, info):
        return self.set_action(self.patp, 'meld','urth', info)

    def stop_running(self, patp):
        # stop pier if running and returns
        # previous run state
        if self.urb_docker.is_running(patp):
            self.broadcast("stopping")
            return self.urb.stop(patp)
        return False

    # Stop devmode if running and returns
    # previous devmode setting
    def stop_devmode(self, patp):
        devmode = self.urb._urbits[patp]['dev_mode'] 
        if devmode:
            dev_mode = False
            self.urb.save_config(patp)
            return True
        return False

    # Revert to devmode to previous setting
    def revert_devmode(self, patp, mode):
        self.urb._urbits[patp]['dev_mode'] = mode
        self.urb.save_config(patp)

    # Revert pier run state
    def revert_running(self, patp, running, start):
        if running:
            res = start(patp, 'boot') == "succeeded"
            return res
        return True

    # Pack or Meld?
    def action(self, patp, act, start):
        if start(patp, act) == act:
            Log.log(f"{patp}: Successfully started container for Urth {act}ing")
            self.broadcast(f"{act}ing")
            while self.urb_docker.is_running(patp):
                sleep(0.5)
            Log.log(f"{patp}: Done Urth {act}ing")
            return True
        else:
            Log.log(f"{patp}: Failed to Urth {act}")
            return False
