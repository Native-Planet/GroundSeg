from time import sleep
from datetime import datetime

from log import Log

class MeldUrth:
    def __init__(self, parent, patp, state):# urb, ws_util):
        self.state = state
        self.broadcaster = self.state['broadcaster']
        self.start = parent.start # start function
        self.get_config = parent.get_config # retrieves value from <patp>.json
        self.set_config = parent.set_config # modify <patp>.json
        self.patp = patp
        self.urb = self.state['dockers']['urbit'] 
        self.urb_docker = self.urb.urb_docker

    def run(self):
        # start is container start command (see ws_urbits.py)
        patp = self.patp
        running = self.stop_running(patp)
        devmode = self.stop_devmode(patp)
        start = self.start

        try:
            if self.action(patp,'pack', start):
                if self.action(patp,'meld', start):
                    self.revert_devmode(patp, devmode)
                    if self.revert_running(patp, running, start):
                        Log.log(f"{patp}: Urth Pack and Meld action completed")
                        self.set_meld_status(patp)
                        self.broadcast("success")
                    else:
                        print("1")
                        raise Exception()
                else:
                    print("2")
                    raise Exception()
            else:
                print("3")
                raise Exception()
        except Exception as e:
            Log.log(f"{patp}: Urth Pack and Meld action did not complete. {e}")
            self.broadcast("failure")
        sleep(3)
        self.broadcast("")

    def broadcast(self, info):
        return self.broadcaster.urbit_broadcast(self.patp, 'meld','urth', info)

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
        # act is either pack or meld
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

    # Set last Meld
    def set_meld_status(self, patp):
        # Get the current time
        now = datetime.utcnow()
        # set current time in as meld_last
        self.set_config(patp, 'meld_last', str(int(now.timestamp())))

        # set next meld schedule
        meld_time = self.get_config(patp, 'meld_time')
        hour = meld_time[0:2]
        minute = meld_time[2:]
        day = self.get_config(patp, 'meld_frequency') * 60 * 60 * 24
        next_time = int(now.replace(hour=int(hour),
                                    minute=int(minute),
                                    second=0
                                    ).timestamp())
        self.set_config(patp, 'meld_next', str(next_time + day))
