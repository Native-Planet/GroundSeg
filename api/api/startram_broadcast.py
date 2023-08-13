class StarTramBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.cfg = self.app.cfg
        self.transition = {
                "register": None,
                "toggle": None
                }

    def display(self):
        return {
                "info": {
                    "registered":self.cfg.system.get('wgRegistered'),
                    "running":self.cfg.system.get('wgOn'),
                    "region":self.app.wireguard.anchor_data.get('region'),
                    "expiry":self.app.wireguard.anchor_data.get('lease'),
                    "renew":self.app.wireguard.anchor_data.get('ongoing') != 0,
                    "endpoint":self.cfg.system.get('endpointUrl'),
                    "regions":self.app.wireguard.region_data,
                    },
                "transition": self.transition
                }

    def set_transition(self,transition,value):
        self.transition[transition] = value

    def clear_transition(self,transition):
        self.set_transition(transition,None)
