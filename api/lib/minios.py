import string
import secrets

from log import Log

class WSMinIOs:
    def __init__(self,state):
        self.state = state
        self.minio = None
        while self.minio == None:
            try:
                self.minio = self.state['dockers']['minio']
            except:
                sleep(0.5)

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
        return self.state['broadcaster'].urbit_broadcast(patp,'minio',action, info)

    def stop(self, patp):
        self.minio.stop(f"minio_{patp}")

    def start(self, patp, pier_config):
        self.minio.start_minio(f"minio_{patp}", pier_config)
