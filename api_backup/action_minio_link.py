import os
import json
from time import sleep
from datetime import datetime

from click_wrapper import Click
from log import Log

class MinIOLink:
    # TODO: remove unlink
    def __init__(self, urb, ws_util, unlink=False):
        self.unlink = unlink
        self.urb = urb
        self.ws_util = ws_util

    def link(self, pier_config, acc, secret, bucket):
        try:
            if not pier_config['click']:
                raise Exception("click unavailable")

            # current ship
            patp = self.patp = pier_config['pier_name']

            # custom url for s3?
            endpoint = "" # unlink
            if not self.unlink:
                endpoint = pier_config['custom_s3_web']
                if len(endpoint) <= 0:
                    endpoint = f"s3.{pier_config['wg_url']}"

            # click set endpoint
            Log.log(f"{patp}: Attempting to link with click")
            self.broadcast("link")
            payload = {
                    "endpoint":endpoint,
                    "acc":acc,
                    "secret":secret,
                    "bucket":bucket
                    }
            if not Click(patp, "s3", self.urb).run(payload):
                self.broadcast("link-legacy")
                if not Click(patp, "s3-legacy",self.urb).run(payload): 
                    raise Exception("poke returning nack")
            self.broadcast("success")
        except Exception as e:
            Log.log(f"WS: {patp} Failed to set endpoint: {e}")
            self.broadcast(f"failure\n{e}")

        # Default
        sleep(3)
        self.broadcast("")

    def broadcast(self, info):
        if self.unlink:
            return self.ws_util.urbit_broadcast(self.patp, 'minio','unlink', info)
        return self.ws_util.urbit_broadcast(self.patp, 'minio','link', info)
