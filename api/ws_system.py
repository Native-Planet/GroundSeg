from action_linux_update import LinuxUpdate

class WSSystem:
    def __init__(self, config, ws_util):
        self.config_object = config
        self.config = config.config
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

        # no            -  unregistered
        # yes           -  a command was sent
        # <reg loading> -  TODO
        # success       -  registered successfully
        # failure\n<err> -  Failure message
        self.ws_util.system_broadcast('system','startram','register','yes')

        # running  -  Wireguard container is running
        # stopped  -  Wireguard container is stopped
        self.ws_util.system_broadcast('system','startram',"container","running")

        self.ws_util.system_broadcast('system','startram',"autorenew",False)
        self.ws_util.system_broadcast('system','startram',"region","us-east")
        self.ws_util.system_broadcast('system','startram',"regions",["us-east"])
        self.ws_util.system_broadcast('system','startram',"expiry",0)
        self.ws_util.system_broadcast('system','startram',"endpoint","api.startram.io")
        self.ws_util.system_broadcast('system','startram',"restart","hide")
        self.ws_util.system_broadcast('system','startram',"cancel","hide")
        self.ws_util.system_broadcast('system','startram',"advanced",False)

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
