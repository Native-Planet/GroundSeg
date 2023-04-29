import string
import secrets

from log import Log

class WSMinIOs:
    def __init__(self, minio, ws_util):
        self.minio = minio
        self.ws_util = ws_util

    def create_account(self, pier_config):
        patp = pier_config['pier_name']
        Log.log(f"WS: {patp}: Attempting to set MinIO endpoint")
        self.broadcast(patp, 'link', 'create-account')
        acc = 'urbit_minio'
        secret = ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(40))

        if self.minio.make_service_account(pier_config, patp, acc, secret):
            return acc, secret
        return False, False

    def broadcast(self, patp, action, info):
        return self.ws_util.urbit_broadcast(patp,'minio',action, info)
