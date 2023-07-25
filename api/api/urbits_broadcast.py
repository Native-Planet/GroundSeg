import socket

class UrbitsBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.cfg = self.app.cfg

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
                if cfg['show_urbit_web'] == 'alias':
                    urb_alias = True
                    url = f"https://{cfg.get('custom_urbit_web')}"
                urbits[str(p)] = {
                        "network": cfg.get('network'),
                        "running": self.app.urbit.urb_docker.is_running(p),
                        "url": url,
                        "urbAlias": urb_alias,
                        "memUsage": self.app.urbit.urb_docker.get_memory_usage(p),
                        "diskUsage": self.app.urbit.urb_docker.get_disk_usage(p),
                        "loomSize": 2 ** (int(cfg.get('loom_size')) - 20),
                        "serviceRegistrationStatus":svc_reg_status
                        }

            except: 
                pass
        return urbits
