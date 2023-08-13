import socket

class UrbitsBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.cfg = self.app.cfg
        self.transition = {}

    def display(self):
        urbits = {}
        for p in self.cfg.system.get('piers').copy():
            try:
                svc_reg_status = "ok"
                try:
                    services = self.app.wireguard.anchor_services.get(p)
                    for svc in services:
                        service = services.get(svc,{"status":"failed"})['status']
                        if service != "ok":
                            svc_reg_status = "creating"
                            break
                except:
                    pass

                cfg = self.app.urbit._urbits[p]
                urb_alias = False
                url = f"http://{socket.gethostname()}.local:{cfg.get('http_port')}"
                if cfg.get('network') == "wireguard":
                    url = f"https://{cfg.get('wg_url')}"
                if cfg['show_urbit_web'] == 'alias':
                    urb_alias = True
                    url = f"https://{cfg.get('custom_urbit_web')}"
                urbits[str(p)] = {
                        "info":{
                            "network": cfg.get('network'),
                            "running": self.app.urbit.urb_docker.is_running(p),
                            "url": url,
                            "urbAlias": urb_alias,
                            "memUsage": self.app.urbit.system_info.get(p),
                            "diskUsage": self.app.urbit.urb_docker.get_disk_usage(p),
                            "loomSize": 2 ** (int(cfg.get('loom_size')) - 20),
                            "devMode": cfg.get('dev_mode'),
                            "detectBootStatus": cfg.get('boot_status') != "off",
                            "remote": cfg.get('network') == "wireguard",
                            "vere":self.app.urbit.vere_version.get(p)
                            },
                        "transition":{
                            "meld": self.get_transition(str(p),"meld"),
                            "serviceRegistrationStatus":svc_reg_status,
                            "togglePower":self.get_transition(str(p),"togglePower"),
                            "deleteShip":self.get_transition(str(p),"deleteShip")
                            }
                        }
            except: 
                pass
        return urbits

    def get_transition(self, patp, transition):
        try:
            return self.transition.get(patp, {}).get(transition)
        except:
            return None

    def set_transition(self, patp, transition, value):
        if not self.transition.get(patp):
            self.transition[patp] = {}
        self.transition[patp][transition] = value

    def clear_transition(self, patp, transition):
        self.set_transition(patp, transition, None)
