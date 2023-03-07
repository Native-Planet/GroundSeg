# Python
from time import sleep

# GroundSeg modules
from log import Log

class DockerUpdater:
    def __init__(self, config, orchestrator):
        self.config_object = config
        self.config = config.config
        self.arch = config._arch
        self.orchestrator = orchestrator
        self.wireguard = orchestrator.wireguard
        self.webui = orchestrator.webui
        self.minio = orchestrator.minio
        self.urbit = orchestrator.urbit
        self.netdata = orchestrator.netdata

    def check_docker_update(self):
        Log.log("Updater: Docker updater thread started")
        while True:
            if self.config['updateMode'] == 'auto':
                try:
                    branch = self.config['updateBranch']
                    self.payload = self.config_object.update_payload['groundseg'][branch]

                    self.config_object.anchor_ready = False
                    Log.log("Anchor: Refresh loop is unready")
                    try:
                        self.update_wireguard()
                    except Exception as e:
                        Log.log(f"Updater: Wireguard update failed: {e}")
                    try:
                        self.update_webui()
                    except Exception as e:
                        Log.log(f"Updater: WebUI update failed: {e}")
                    try:
                        self.update_mc()
                    except Exception as e:
                        Log.log(f"Updater: MinIO Client update failed: {e}")
                    try:
                        self.update_minio()
                    except Exception as e:
                        Log.log(f"Updater: MinIO update failed: {e}")
                    try:
                        self.update_urbit()
                    except Exception as e:
                        Log.log(f"Updater: Urbit update failed: {e}")
                    try:
                        self.update_netdata()
                    except Exception as e:
                        Log.log(f"Updater: Netdata update failed: {e}")

                    Log.log("Anchor: Refresh loop is ready")
                    self.config_object.anchor_ready = True
                    sleep(self.config['updateInterval'])
                except Exception as e:
                    Log.log(f"Updater: Docker updater failed: {e}")
                    sleep(60)

    def update_wireguard(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log(f"Updater: Checking for wireguard updates")
            info = self.payload['wireguard']
            wg_name = self.wireguard.data['wireguard_name']
            tag = self.wireguard.data['wireguard_version']
            repo = info['repo']

            if tag == "latest" or tag == "edge":
                sha = f"{self.arch}_sha256"
                image = f"{repo}:tag@sha256:{info[sha]}"
            else:
                image = f"{repo}:{tag}"

            c = self.wireguard.wg_docker.get_container(wg_name)
            if c:
                old_image = c.attrs['Config']['Image']
                if old_image != image:
                    Log.log(f"Updater: Wireguard update detected. Updating..")
                    try:
                        remote = []
                        for patp in self.urbit._urbits:
                            if self.urbit._urbits[patp]['network'] != 'none':
                                remote.append(patp)

                        if self.wireguard.off(self.urbit, self.minio) == 200:
                            if self.wireguard.remove():
                                if self.wireguard.on(self.minio) == 200:
                                    if len(remote) > 0:
                                        for patp in remote:
                                            if self.urbit.toggle_network(patp) == 200:
                                                Log.log(f"Updater: Wireguard update complete")
                    except Exception as e:
                        Log.log(f"Updater: Failed to update wireguard: {e}")
                else:
                    Log.log(f"Updater: Wireguard already on correct version")

    def update_webui(self):
        name = self.webui.data['webui_name']
        tag = self.webui.data['tag']
        info = self.payload['webui']
        if tag == "latest" or tag == "edge":
            sha = f"{self.arch}_sha256"
            image = f"{info['repo']}:tag@sha256:{info[sha]}"
        else:
            image = f"{updater_info['repo']}:{tag}"
        c = self.webui.webui_docker.get_container(name)
        if c:
            old_image = c.attrs['Config']['Image']
            if old_image != image:
                Log.log(f"Updater: WebUI update detected. Updating..")
                if self.webui.start():
                    Log.log(f"Updater: WebUI update complete")
            else:
                Log.log("Updater: WebUI already correct version")

    def update_netdata(self):
        name = self.netdata.data['netdata_name']
        tag = self.netdata.data['netdata_version']
        info = self.payload['netdata']
        if tag == "latest" or tag == "edge":
            sha = f"{self.arch}_sha256"
            image = f"{info['repo']}:tag@sha256:{info[sha]}"
        else:
            image = f"{updater_info['repo']}:{tag}"
        c = self.netdata.nd_docker.get_container(name)
        if c:
            old_image = c.attrs['Config']['Image']
            if old_image != image:
                Log.log(f"Updater: Netdata update detected. Updating..")
                if self.netdata.start():
                    Log.log(f"Updater: Netdata update complete")
            else:
                Log.log("Updater: Netdata already correct version")

    def update_mc(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log(f"Updater: Checking for MinIO Client updates")
            info = self.payload['miniomc']
            sha = f"{self.arch}_sha256"
            image = f"{info['repo']}:tag@sha256:{info[sha]}"
            mc_name = self.minio.mc_name
            c = self.minio.mc_docker.get_container(mc_name)
            if c:
                old_image = c.attrs['Config']['Image']
                if old_image != image:
                    Log.log(f"Updater: MinIO Client update detected. Updating..")
                    try:
                        if self.minio.mc_docker.remove_container(mc_name):
                            if self.minio.start_mc():
                                Log.log(f"Updater: MinIO Client update complete")
                    except Exception as e:
                        Log.log(f"Updater: Failed to update MinIO Client: {e}")
                else:
                    Log.log(f"Updater: MinIO Client already on correct version")


    def update_minio(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log(f"Updater: Checking for MinIO updates")
            copied = self.urbit._urbits
            for p in list(copied):
                info = self.payload['minio']
                name = f"minio_{p}"
                sha = f"{self.arch}_sha256"
                if self.urbit._urbits[p]['minio_password'] != '':
                    Log.log(f"{name}: Checking for MinIO update")
                    c = self.minio.minio_docker.get_container(name)
                    tag = self.urbit._urbits[p]['minio_version']
                    if tag == "latest" or tag == "edge":
                        sha = f"{self.arch}_sha256"
                        image = f"{info['repo']}:tag@sha256:{info[sha]}"
                    else:
                        image = f"{info['repo']}:{tag}"
                    if c:
                        old_image = c.attrs['Config']['Image']
                        if old_image != image:
                            Log.log(f"{name}: MinIO update detected. Updating..")
                            try:
                                if self.minio.minio_docker.remove_container(name):
                                    if self.minio.start_minio(name, self.urbit._urbits[p]):
                                        Log.log(f"{name}: MinIO update complete")
                            except Exception as e:
                                Log.log(f"{name}: Failed to update MinIO: {e}")
                        else:
                            Log.log(f"{name}: MinIO already on correct version")


    def update_urbit(self):
        Log.log(f"Updater: Checking for Urbit updates")
        copied = self.urbit._urbits
        for p in list(copied):
            info = self.payload['vere']
            sha = f"{self.arch}_sha256"
            Log.log(f"{p}: Checking for Urbit update")
            tag = self.urbit._urbits[p]['urbit_version']
            if tag == "latest" or tag == "edge":
                sha = f"{self.arch}_sha256"
                image = f"{info['repo']}:tag@sha256:{info[sha]}"
            else:
                image = f"{info['repo']}:{tag}"
            c = self.urbit.urb_docker.get_container(p)
            if c:
                old_image = c.attrs['Config']['Image']
                if old_image != image:
                    Log.log(f"{p}: Urbit update detected. Updating..")
                    try:
                        if self.urbit.urb_docker.remove_container(p):
                            if self.urbit.start(p) == "succeeded":
                                Log.log(f"{p}: Urbit update complete")
                    except Exception as e:
                        Log.log(f"{p}: Failed to update Urbit: {e}")
                else:
                    Log.log(f"{p}: Urbit already on correct version")
