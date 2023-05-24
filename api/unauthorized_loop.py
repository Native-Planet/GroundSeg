from time import sleep
from datetime import datetime, timedelta

from log import Log

class UnauthorizedLoop:
    def __init__(self, config, ws_util):
        self.config_object = config
        self.config = config.config
        self.ws_util = ws_util
        self.count = 0

    def run(self):
        Log.log("unauthorized_loop Starting thread")
        self.ws_util.system_broadcast('system','login','access','unauthorized')
        while True:
            # Cleanup unauth
            self.clean_unauthorized()

            # note that "access" is not set here
            # this is done right before the message is sent
            self._attempts()
            self._cooldown()
            self._status()
            self.count += 1
            sleep(1)

    def clean_unauthorized(self):
        # check if past 5 minutes
        sessions = self.config['sessions']['unauthorized'].copy()
        print(sessions)
        for token in sessions:
            created = sessions[token]['created']
            expire = datetime.strptime(created, "%Y-%m-%d_%H:%M:%S") + timedelta(minutes=1)
            print(token)
            print("created ",created)
            print("expire ", expire)
            now = datetime.now()
            if now >= expire:
                # remove from config
                self.config['sessions']['unauthorized'].pop(token)
                # close the user's connection
                for websocket in self.ws_util.unauthorized_clients:
                    print(token)
                    print(self.ws_util.unauthorized_clients[websocket])
                #await websocket.close(code=1000, reason="unauthorized session expired")


    #
    #   Login
    #

    def _attempts(self):
        try:
            a = self.config_object.login_status['attempts']
        except Exception as e:
            Log.log(f"unauthorized_loop:_attempts {e}")
            a = f"error\n{e}"
        self.ws_util.system_broadcast('system','login','attempts', a)

    def _cooldown(self):
        c = 0
        try:
            end = self.config_object.login_status['end']
            now = datetime.now()
            if end > now:
                c = int((end - now).total_seconds())
        except:
            pass
        self.ws_util.system_broadcast('system','login','cooldown', c)

    #
    #   Setup
    #

    def _status(self):
        # TODO: make this work
        self.ws_util.system_broadcast('system','setup','status', 'done')
