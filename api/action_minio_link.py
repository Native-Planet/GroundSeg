import os
import json
from time import sleep
from datetime import datetime

from log import Log

class MinIOLink:
    def __init__(self, parent, urb):
        self.urb = urb #for %lens
        self.set_action = parent.set_action

    def link(self, pier_config, acc, secret, bucket):
        try:
            # current ship
            self.patp = patp = pier_config['pier_name']

            # custom url for s3?
            endpoint = pier_config['custom_s3_web']
            if len(endpoint) <= 0:
                endpoint = f"s3.{pier_config['wg_url']}"

            # click set endpoint
            Log.log(f"{patp}: Attempting to link with click")
            self.broadcast("link-click")
            sleep(3)
            click = False
            Log.log(f"{patp}: Failed to link with click")
            # clicked failed, try lens
            if not click:
                try:
                    addr = self.urb.get_loopback_addr(patp)
                    self.broadcast("link-lens")
                    Log.log(f"{patp}: Attempting to link with %lens")
                    self.set_endpoints(patp, endpoint, acc, secret, bucket, addr)
                except Exception as e:
                    Log.log(f"{patp}: Failed to link with %lens")
                    raise Exception("%lens failed: {e}")

            self.broadcast("success")
        except Exception as e:
            Log.log(f"WS: {patp} Failed to set endpoint: {e}")
            self.broadcast("failure")

        sleep(3)
        self.broadcast("")

    def broadcast(self, info):
        return self.set_action(self.patp, 'minio','link', info)

    def set_endpoints(self, patp, endpoint, access_key, secret, bucket, lens_addr):
        self.lens_poke(patp, 'set-endpoint', endpoint, lens_addr)
        self.lens_poke(patp, 'set-access-key-id', access_key, lens_addr)
        self.lens_poke(patp, 'set-secret-access-key', secret, lens_addr)
        self.lens_poke(patp, 'set-current-bucket', bucket, lens_addr)
    '''
    def unlink_minio_endpoint(self, patp, lens_addr):
        self.send_poke(patp, 'set-endpoint', '', lens_addr)
        self.send_poke(patp, 'set-access-key-id', '', lens_addr)
        self.send_poke(patp, 'set-secret-access-key', '', lens_addr)
        self.send_poke(patp, 'set-current-bucket', '', lens_addr)

        return 200
    '''

    def lens_poke(self, patp, command, data, lens_addr):
        Log.log(f"{patp}: Attempting to send {command} poke")
        try:
            data = {"source": {"dojo": f"+landscape!s3-store/{command} '{data}'"}, "sink": {"app": "s3-store"}}
            with open(f'{self.urb._volume_directory}/{patp}/_data/{command}.json','w') as f :
                json.dump(data, f)

            cmd = f'curl -s -X POST -H "Content-Type: application/json" -d @{command}.json {lens_addr}'
            res = self.urb.urb_docker.exec(patp, cmd)
            if res:
                os.remove(f'{self.urb._volume_directory}/{patp}/_data/{command}.json')
                Log.log(f"{patp}: {command} sent successfully")

        except Exception as e:
            Log.log(f"{patp}: Failed to send {command}: {e}")
