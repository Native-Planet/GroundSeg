
class ResponseBuilder:
    def __init__(self, cfg, orchestrator):
        # config class
        # for system.json dict us self.cfg.config
        self.cfg = cfg
        self.orc = orchestrator

    def client_dump(self):
        message = {
                "system": None,
                "urbits": self.build_urbits()
                }
        return message

    def build_urbits(self):
        res = {}
        for patp in self.cfg.config['piers']:
            res[patp] = {}
            '''
            running = self.orc.urbit.urb_docker.is_running(patp)
            booted = len(self.orc.urbit.get_code(patp)) == 27
            if not running:
                res[patp]['status'] = "stopped"
            elif not booted:
                res[patp]['status'] = "booting"
            else:
                res[patp]['status'] = "running"
            '''
        return res
