# Python
import os
import json
import zipfile
from io import BytesIO
from time import sleep

# Flask
from flask import send_file

# GroundSeg modules
from log import Log
from mc_docker import MCDocker
from minio_docker import MinIODocker

class MinIO:
    mc_data = {}
    updater_mc = {}
    default_mc_config = {
            "mc_name": "minio_client",
            "mc_version": "latest",
            "repo": "registry.hub.docker.com/minio/mc",
            "amd64_sha256": "6ffd76764e8ca484de12c6ecaa352db3d8efd5c9d44f393718b29b6600e0a559",
            "arm64_sha256": "6825aecd2f123c9d4408e660aba8a72f9e547a3774350b8f4d2d9b674e99e424"
            }
    minios_on = False

    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self, config, wg):
        self.config_object = config
        self.config = config.config
        self.wg = wg
        self.filename = f"{self.config_object.base_path}/settings/mc.json"
        self._volume_directory = f"{self.config['dockerData']}/volumes"
        self.mc_docker = MCDocker()
        self.minio_docker = MinIODocker()

        # Set MC Config
        self.load_config()
        branch = self.config['updateBranch']
        self.mc_data = {**self.default_mc_config, **self.mc_data}

        # Updater MC information
        if (self.config_object.update_avail) and (self.config['updateMode'] == 'auto'):
            Log.log("MC: Replacing local data with version server data")
            self.updater_mc = self.config_object.update_payload['groundseg'][branch]['miniomc']
            self.mc_data['repo'] = self.updater_mc['repo']
            self.mc_data['mc_version'] = self.updater_mc['tag']
            self.mc_data['amd64_sha256'] = self.updater_mc['amd64_sha256']
            self.mc_data['arm64_sha256'] = self.updater_mc['arm64_sha256']

        self.save_config()

        if self.config['wgOn'] and self.config['wgRegistered']:
            self.start_mc()
            sleep(3)
            self.start_all()

        Log.log("MinIO: Initialization Completed")

    # Create MinIO
    def create_minio(self, patp, password, urb, link):
        Log.log(f"{patp}: Attempting to create MinIO")
        try:
            urb._urbits[patp]['minio_password'] = password
            urb.save_config(patp)
            if self.start_minio(f"minio_{patp}", urb._urbits[patp]):
                if not link:
                    return 200
                if urb.set_minio(patp) == 200:
                    return 200

        except Exception as e:
            Log.log(f"{patp}: Failed to create MinIO: {e}")

        return 400

    def start_mc(self):
        return self.mc_docker.start(self.mc_data, self.config_object._arch)

    def start_minio(self, name, pier_config):
        if self.config['wgOn'] and self.config['wgRegistered'] and pier_config['minio_password'] != '':
            if self.minio_docker.start(name, pier_config, self.config_object._arch):
                return self.mc_setup(name, pier_config)
        # Skip
        return True

    def start_all(self):
        self.minios_on = self.minio_docker.start_all()
        return self.minios_on

    def stop_all(self):
        self.minios_on = False
        return self.minio_docker.stop_all()

    def stop_mc(self):
        return self.mc_docker.stop(self.mc_data['mc_name'])

    def stop_minio(self, name):
        return self.minio_docker.stop(name)
    
    def delete(self, name):
        return self.minio_docker.delete(name)

    def export(self, patp):
        name = f"minio_{patp}"
        Log.log(f"{name}: Attempting to export bucket")
        c = self.minio_docker.get_container(name)
        if c:
            file_name = f"bucket_{patp}.zip"
            memory_file = BytesIO()
            file_path=f"{self._volume_directory}/{name}/_data/bucket"

            Log.log(f"{name}: Compressing bucket")

            with zipfile.ZipFile(memory_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
                for root, dirs, files in os.walk(file_path):
                    arc_dir = root[root.find("_data/")+6:]
                    for file in files:
                        zipf.write(os.path.join(root, file), arcname=os.path.join(arc_dir,file))

            memory_file.seek(0)

            Log.log(f"{patp}: Pier successfully exported")
            return send_file(memory_file, download_name=file_name, as_attachment=True)

    def mc_setup(self, name, pier_config):
        Log.log(f"{name}: Attempting to create MinIO admin account")
        try:
            patp = pier_config['pier_name']
            port = pier_config['wg_s3_port']
            pwd = pier_config['minio_password']
            self.mc_docker.exec(self.mc_data['mc_name'], f"mc alias set patp_{patp} http://localhost:{port} {patp} {pwd}")
            self.mc_docker.exec(self.mc_data['mc_name'], f"mc anonymous set public patp_{patp}/bucket")
            Log.log(f"{name}: Created MinIO admin account")
            return True

        except Exception as e:
            Log.log(f"{name}: Failed to create MinIO admin account: {e}")

        return False

    def make_service_account(self, pier_config, patp, acc, pwd):
        x = None
        name = f"minio_{patp}"

        Log.log(f"{name}: Attempting to make service account")
        try:
            # create admin account if failed previously
            if self.mc_setup(name, pier_config):
                c = self.mc_docker.get_container(self.mc_data['mc_name'])
                if c:
                    Log.log(f"{name}: Attempting to update service account credentials.")
                    command = f"mc admin user svcacct edit --secret-key '{pwd}' patp_{patp} {acc}"
                    x = c.exec_run(command, tty=True).output.decode('utf-8').strip()

                    if 'ERROR' in x:
                        Log.log(f"{name}: Service account does not exist. Creating new account")
                        command = f"mc admin user svcacct add --access-key '{acc}' --secret-key '{pwd}' patp_{patp} {patp}"
                        x = c.exec_run(command).output.decode('utf-8').strip()

                        if 'ERROR' in x:
                            raise Exception(x)

                    Log.log(f"{name}: Service account created")
                    return True

        except Exception as e:
            Log.log(f"{name}: Failed to update service account credentials: {e}")

        return False

    # Container logs
    def minio_logs(self, name):
        return self.minio_docker.full_logs(name)

    # Load mc.json
    def load_config(self):
        try:
            with open(self.filename) as f:
                self.mc_data = json.load(f)
                Log.log("MC: Successfully loaded mc.json")

        except Exception as e:
            Log.log(f"MC: Failed to open mc.json: {e}")
            Log.log("MC: New mc.json will be created")

    # Save mc.json
    def save_config(self):
        with open(self.filename,'w') as f:
            json.dump(self.mc_data, f, indent=4)
            f.close()
