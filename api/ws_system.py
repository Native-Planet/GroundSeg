from threading import Thread
from action_linux_update import LinuxUpdate

class WSSystem:
    def __init__(self, config, wg, ws_util):
        self.config_object = config
        self.config = config.config
        self.wg = wg
        self.ws_util = ws_util

        # {a:{b:{c:d}}
        self.ws_util.system_broadcast('updates','linux','upgrade','0')
        self.ws_util.system_broadcast('updates','linux','new','0')
        self.ws_util.system_broadcast('updates','linux','remove','0')
        self.ws_util.system_broadcast('updates','linux','ignore','0')

        if self.config['linuxUpdates']['previous']:
            # updated       -  no updates
            # initializing  -  a command was sent
            # command       -  running apt upgrade -y
            # restarting    -  update complete, restarting device
            # success       -  GroundSeg has restarted
            # failure\n<err> -  Failure message
            self.ws_util.system_broadcast('updates','linux','update','success')
            self.config['linuxUpdates']['previous'] = False
            self.config_object.save_config()

        # TODO
        self.ws_util.system_broadcast('updates','binary','update','updated')
        self.ws_util.system_broadcast('updates','binary','routine','auto')       # notify off

        from unauthorized_loop import UnauthorizedLoop
        #import asyncio
        _unauthorized = UnauthorizedLoop(self.config_object, self.ws_util)
        #asyncio.run(_unauthorized.run())
        Thread(target=_unauthorized.run, daemon=True).start()
        '''

        from startram_loop import StarTramLoop
        startram = StarTramLoop(self.config_object, self.wg, self.ws_util)
        Thread(target=startram.run, daemon=True).start()
        '''


    #
    #   Actions
    #

    def linux_update(self):
        old_info = "pending"
        try:
            old_info = self.ws_util.structure['updates']['linux']['update']
        except:
            pass
        self.ws_util.system_broadcast('updates', 'linux','update','initializing')
        LinuxUpdate(self.ws_util, self.config_object).run(old_info)
