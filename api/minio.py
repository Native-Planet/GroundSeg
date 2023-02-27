# GroundSeg modules
from log import Log
from mc_docker import MCDocker
from minio_docker import MinIODocker

class MinIO:
    data = {}
    updater_mc = {}
    updater_minio = {}
    mc_name = "minio_client"

    def __init__(self, config, wg):
        self.config_object = config
        self.config = config.config
        self.wg = wg
        self.mc_docker = MCDocker()
        self.minio_docker = MinIODocker()

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("MinIO: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater MC information
        if self.config_object.update_avail:
            self.updater_mc = self.config_object.update_payload['groundseg'][branch]['miniomc']
            self.updater_minio = self.config_object.update_payload['groundseg'][branch]['minio']

        if self.config['wgOn'] and self.config['wgRegistered']:
            self.start_mc()

        Log.log("MinIO: Initialization Completed")

    # Create MinIO
    def create_minio(self, patp, password, urb):
        Log.log(f"{patp}: Attempting to create MinIO")
        try:
            urb._urbits[patp]['minio_password'] = password
            urb.save_config(patp)
            if self.start_minio(f"minio_{patp}", urb._urbits[patp]):
                return 200
        except Exception as e:
            Log.log(f"{patp}: Failed to create MinIO: {e}")

        return 400

    def start_mc(self):
        return self.mc_docker.start(self.mc_name, self.updater_mc, self.config_object._arch)

    def start_minio(self, name, pier_config):
        if self.config['wgOn'] and self.config['wgRegistered'] and pier_config['minio_password'] != '':
            if self.minio_docker.start(name, self.updater_minio, pier_config, self.config_object._arch):
                return self.mc_setup(name, pier_config)
        # Skip
        return True

    def stop_minio(self, name):
        return self.minio_docker.stop(name)
    
    def delete(self, name):
        return self.minio_docker.delete(name)

    def mc_setup(self, name, pier_config):
        Log.log(f"{name}: Attempting to create MinIO admin account")
        try:
            patp = pier_config['pier_name']
            port = pier_config['wg_s3_port']
            pwd = pier_config['minio_password']
            self.mc_docker.exec(self.mc_name, f"mc alias set patp_{patp} http://localhost:{port} {patp} {pwd}")
            self.mc_docker.exec(self.mc_name, f"mc anonymous set public patp_{patp}/bucket")
            Log.log(f"{name}: Created MinIO admin account")
            return True

        except Exception as e:
            Log.log(f"{name}: Failed to create MinIO admin account: {e}")

        return False

    '''
    def make_service_account(self, patp, acc, pwd):
        x = None

        print('Updating service account credentials', file=sys.stderr)
        x = self.container.exec_run(f"mc admin user svcacct edit \
                --secret-key '{pwd}' \
                patp_{patp} {acc}", tty=True).output.decode('utf-8').strip()

        if 'ERROR' in x:
            print('Service account does not exist. Creating new account...', file=sys.stderr)
            x = self.container.exec_run(f"mc admin user svcacct add \
                    --access-key '{acc}' \
                    --secret-key '{pwd}' \
                    patp_{patp} {patp}").output.decode('utf-8').strip()
        
            if 'ERROR' in x:
                return 400

        return 200
        
    '''
