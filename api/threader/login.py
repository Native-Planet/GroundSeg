from datetime import datetime
from time import sleep

class LoginLoop:
    def __init__(self, state): 
        self.state = state
        self.broadcaster = self.state['broadcaster']

        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

    def run(self):
        self._attempts()
        self._cooldown()

    def _attempts(self):
        try:
            a = self.config_object.login_status['attempts']
        except Exception as e:
            print(f"threader:login:_attempts {e}")
            a = f"error\n{e}"
        self.broadcaster.system_broadcast('system','login','attempts', a)

    def _cooldown(self):
        c = 0
        try:
            end = self.config_object.login_status['end']
            now = datetime.now()
            if end > now:
                c = int((end - now).total_seconds())
        except:
            pass
        self.broadcaster.system_broadcast('system','login','cooldown', c)
