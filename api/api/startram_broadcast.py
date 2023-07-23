class StarTramBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.cfg = self.app.cfg
        self.data = self.app.wireguard.anchor_data

    def display(self):
        return {
                "registered":self.cfg.system.get('wgRegistered'),
                "running":self.cfg.system.get('wgOn'),
                "region":self.data.get('region'),
                "expiry":self.data.get('lease'),
                "renew":self.data.get('ongoing') != 0,
                "endpoint":self.cfg.system.get('endpointUrl'),
                "regions":self.app.wireguard.region_data,
                "registerStatus":"not null" #self.app.wireguard.register_broadcast_status
                }
