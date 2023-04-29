class WSSystem:
    def __init__(self, ws_util):
        self.ws_util = ws_util

        # {'updates':{'linux':{x,y}}
        self.linux_broadcast('update','pending') # updated updating <error>
        self.linux_broadcast('upgrade','0')
        self.linux_broadcast('new','0')
        self.linux_broadcast('remove','0')
        self.linux_broadcast('ignore','0')

        # {'updates':{'binary':{x,y}}
        self.binary_broadcast('update','updated') # updating pending
        self.binary_broadcast('routine','auto') # notify off

        # {'system':{'startram':{x,y}}
        self.startram_broadcast("container","stopped") # running
        self.startram_broadcast("autorenew",False)
        self.startram_broadcast("region","us-east")
        self.startram_broadcast("expiry",0)
        self.startram_broadcast("endpoint","api.startram.io")
        self.startram_broadcast("register","no")
        self.startram_broadcast("restart","hide")
        self.startram_broadcast("cancel","hide")
        self.startram_broadcast("advanced",False)

    def linux_broadcast(self, action, info):
        self.ws_util.system_broadcast('updates','linux',action,info)

    def binary_broadcast(self, action, info):
        self.ws_util.system_broadcast('updates','binary',action,info)

    def startram_broadcast(self, action, info):
        self.ws_util.system_broadcast('system','startram',action,info)
