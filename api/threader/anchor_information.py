# Python
from  time import sleep
from threading import Thread

# GroundSeg modules
from log import Log

class AnchorInformation:
    def __init__(self,state):#config, orchestrator):
        self.state = state

        # Config
        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config

        # StarTram API
        self.api = None
        while self.api == None:
            try:
                self.api = self.state['startram']
            except:
                pass
            sleep(2)

        self.count = 0
    # Get updated Anchor information every 12 hours
    def run(self):
        n = ((60 * 60 * 6) - 60) 
        if (self.count == 0) or ((self.count % n) == 0):
            Thread(target=self.api.get_regions, args=(30,),daemon=True).start()
            if self.config['wgRegistered'] and self.config['wgOn']:
                try:
                    if self.orchestrator.startram_api.retrieve_status():
                        wg_conf = self.orchestrator.wireguard.anchor_data['conf']
                        if self.orchestrator.wireguard.update_wg_config(wg_conf):
                            self.update_urbit()
                except Exception as e:
                    Log.log(f"anchor_information:run Failed to get updated anchor information: {e}")

        self.count += 1

    '''
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
    '''
