from threading import Thread
#from action_linux_update import LinuxUpdate

class WSSystem:
    def __init__(self, state):
        self.state = state
        self.config_object = self.state['config']
        self.broadcaster = self.state['broadcaster']

        while self.config_object == None:
            print(self.config_object)
            sleep(0.5)
            self.config_object = self.state['config']

        self.config = self.config_object.config

        # {a:{b:{c:d}}
        self.broadcaster.system_broadcast('updates','linux','upgrade','0')
        self.broadcaster.system_broadcast('updates','linux','new','0')
        self.broadcaster.system_broadcast('updates','linux','remove','0')
        self.broadcaster.system_broadcast('updates','linux','ignore','0')

        if self.config['linuxUpdates']['previous']:
            # updated       -  no updates
            # initializing  -  a command was sent
            # command       -  running apt upgrade -y
            # restarting    -  update complete, restarting device
            # success       -  GroundSeg has restarted
            # failure\n<err> -  Failure message
            self.broadcaster.system_broadcast('updates','linux','update','success')
            self.config['linuxUpdates']['previous'] = False
            self.config_object.save_config()

        # TODO
        self.broadcaster.system_broadcast('updates','binary','update','updated')
        self.broadcaster.system_broadcast('updates','binary','routine','auto')       # notify off

    #
    #   Actions
    #

    def linux_update(self):
        old_info = "pending"
        try:
            old_info = self.state['broadcast']['updates']['linux']['update']
        except:
            pass
        self.state['broadcaster'].system_broadcast('updates', 'linux','update','initializing')
        #LinuxUpdate(self.ws_util, self.config_object).run(old_info)
