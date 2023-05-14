# Python
import time

# GroundSeg modules
from log import Log

class AnchorUpdater:
    def __init__(self, config, orchestrator):
        self.config_object = config
        self.config = config.config
        self.orchestrator = orchestrator

    # Get updated Anchor information every 12 hours
    def anchor_loop(self):
        Log.log("anchor_information:anchor_loop Thread started")
        while True:
            self.orchestrator.startram_api.get_regions()
            if self.config['wgRegistered'] and self.config['wgOn']:
                try:
                    if self.orchestrator.startram_api.retrieve_status():
                        wg_conf = self.orchestrator.wireguard.anchor_data['conf']
                        if self.orchestrator.wireguard.update_wg_config(wg_conf):
                            if self.update_urbit():
                                time.sleep((60 * 60 * 12) - 60)

                except Exception as e:
                    Log.log(f"anchor_information:anchor_loop Failed to get updated anchor information: {e}")

            time.sleep(60)

    def update_urbit(self):
        try:
            for patp in self.orchestrator.urbit._urbits:
                svc_url = None
                http_port = None
                http_alias = None
                ames_port = None
                s3_port = None
                console_port = None
                pub_url = '.'.join(self.config['endpointUrl'].split('.')[1:])

                for ep in self.orchestrator.wireguard.anchor_data['subdomains']:
                    if ep['status'] == 'ok':
                        if f'{patp}.{pub_url}' == ep['url']:
                            svc_url = ep['url']
                            http_port = ep['port']
                            http_alias = ep['alias']
                        elif f'ames.{patp}.{pub_url}' == ep['url']:
                            ames_port = ep['port']
                        elif f'bucket.s3.{patp}.{pub_url}' == ep['url']:
                            s3_port = ep['port']
                        elif f'console.s3.{patp}.{pub_url}' == ep['url']:
                            console_port = ep['port']

                if None not in [svc_url, http_port, ames_port, s3_port, console_port, http_alias]:
                    if not self.orchestrator.urbit.update_wireguard_network(
                            patp,
                            svc_url,
                            http_port,
                            ames_port,
                            s3_port,
                            console_port,
                            http_alias):
                        raise Exception("Unable to update wireguard network")
            return True
        except Exception as e:
            Log.log(f"Anchor: Failed to update urbit wireguard information: {e}")
            return False
