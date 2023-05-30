from threading import Thread
from action_linux_update import LinuxUpdate

class System:
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

        '''

        from startram_loop import StarTramLoop
        startram = StarTramLoop(self.config_object, self.wg, self.ws_util)
        Thread(target=startram.run, daemon=True).start()
        '''

    # Broadcast action for System and Updates
    def system_broadcast(self, category, module, action, info=""):
        try:
            # Whitelist of categories allowed to broadcast
            whitelist = ['system','updates']
            if category not in whitelist:
                raise Exception(f"Error. Category '{category}' not in whitelist")

            # Category
            try:
                if not self.structure.get(category) or not isinstance(self.structure[category], dict):
                    self.structure[category] = {}
            except Exception as e:
                raise Exception(f"failed to set category '{category}': {e}") 

            # Module
            try:
                if not self.structure[category].get(module) or not isinstance(self.structure[category], dict):
                    self.structure[category][module] = {}
            except Exception as e:
                raise Exception(f"failed to set module '{module}': {e}") 

            # Action
            self.structure[category][module][action] = info

        except Exception as e:
            print(f"ws-util:system-broadcast {e}")
            return False

        return True

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
