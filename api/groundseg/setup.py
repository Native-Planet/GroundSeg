import asyncio

class Setup:
    def __init__(self,parent,cfg):
        self.active_sessions = parent.active_sessions
        self.cfg = cfg
        self.stages = {
                "start":0,
                "profile":1,
                "startram":2,
                "complete":3
                }
        self.stage = self.cfg.system.get('setup')
        self.page = self.stages.get(self.stage)

    def begin(self):
        if self.page == 0:
            self.page = 1
            self.stage = "profile"
            self.cfg.system['setup'] = self.stage
            self.cfg.save_config()

    def password(self,pwd):
        if self.page == 1:
            if self.cfg.create_password(pwd):
                self.page = 2
                self.stage = "startram"
                self.cfg.system['setup'] = self.stage
                self.cfg.save_config()

    def complete(self, websocket):
        if self.page == 2:
            # First we remove all the active sessions
            # from both auth and unauth dicts
            # Clearing unauthorized active sessions
            self.active_sessions['unauthorized'].clear()
            # Keep the current session
            tid = self.active_sessions['authorized'][websocket]
            # Clear authorized active sessions
            self.active_sessions['authorized'].clear()
            # Add current session
            self.active_sessions['authorized'][websocket] = tid

            # Next we do the same for the config file
            # Clearing unauthorized sessions from config
            self.cfg.system['sessions']['unauthorized'].clear()
            # Keep the current session
            token_data = self.cfg.system['sessions']['authorized'][tid]
            self.cfg.system['sessions']['authorized'].clear()
            self.cfg.system['sessions']['authorized'][tid] = token_data

            # Finally, we set GroundSeg to complete the setup process
            self.page = 3
            self.stage = "complete"
            self.cfg.system['setup'] = self.stage
            # save config
            self.cfg.save_config()
