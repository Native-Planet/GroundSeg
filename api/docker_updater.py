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
        self.wireguard = self.orchestrator.wireguard
        self.webui = self.orchestrator.webui
        self.minio = self.orchestrator.minio
        self.urbit = self.orchestrator.urbit
        self.netdata = self.orchestrator.netdata

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
            else:
                sleep(60)

    def update_wireguard(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log("Updater: Checking for Wireguard updates")

            # Update payload
            srv = self.payload['wireguard'] 

            # Local info
            loc = self.wireguard.data

            # Modify if changed
            changed = False
            if srv['repo'] != loc['repo']:
                Log.log(f"Updater: Wireguard repo: {loc['repo']} -> {srv['repo']}")
                loc['repo'] = srv['repo']
                changed = True

            if srv['tag'] != loc['wireguard_version']:
                Log.log(f"Updater: Wireguard tag: {loc['wireguard_version']} -> {srv['tag']}")
                loc['wireguard_version'] = srv['tag']
                changed = True

            sha = f"{self.arch}_sha256"
            if srv[sha] != loc[sha]:
                Log.log(f"Updater: Wireguard {sha} {loc[sha]} -> {srv[sha]}")
                loc[sha] = srv[sha]
                changed = True

            if changed:
                try:
                    Log.log("Updater: Wireguard update detected. Updating..")
                    # Save new config
                    self.wireguard.save_config()

                    # List ships that are in remote
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
                                            Log.log("Updater: Wireguard update complete")

                except Exception as e:
                    Log.log(f"Updater: Failed to update wireguard: {e}")
            else:
                Log.log("Updater: Wireguard already on correct version")

    def update_webui(self):
        # Update payload
        srv = self.payload['webui'] 

        # Local info
        loc = self.webui.data

        # Modify if changed
        changed = False
        if srv['repo'] != loc['repo']:
            Log.log(f"Updater: WebUI repo: {loc['repo']} -> {srv['repo']}")
            loc['repo'] = srv['repo']
            changed = True

        if srv['tag'] != loc['webui_version']:
            Log.log(f"Updater: WebUI tag: {loc['webui_version']} -> {srv['tag']}")
            loc['webui_version'] = srv['tag']
            changed = True

        sha = f"{self.arch}_sha256"
        if srv[sha] != loc[sha]:
            Log.log(f"Updater: WebUI {sha} {loc[sha]} -> {srv[sha]}")
            loc[sha] = srv[sha]
            changed = True

        if changed:
            Log.log("Updater: WebUI update detected. Updating..")
            # Save new config
            self.webui.save_config()
            if self.webui.start():
                Log.log("Updater: WebUI update complete")
        else:
            Log.log("Updater: WebUI already correct version")

    def update_netdata(self):
        # Update payload
        srv = self.payload['netdata'] 

        # Local info
        loc = self.netdata.data

        # Modify if changed
        changed = False
        if srv['repo'] != loc['repo']:
            Log.log(f"Updater: Netdata repo: {loc['repo']} -> {srv['repo']}")
            loc['repo'] = srv['repo']
            changed = True

        if srv['tag'] != loc['netdata_version']:
            Log.log(f"Updater: Netdata tag: {loc['netdata_version']} -> {srv['tag']}")
            loc['netdata_version'] = srv['tag']
            changed = True

        sha = f"{self.arch}_sha256"
        if srv[sha] != loc[sha]:
            Log.log(f"Updater: Netdata {sha} {loc[sha]} -> {srv[sha]}")
            loc[sha] = srv[sha]
            changed = True

        if changed:
            Log.log("Updater: Netdata update detected. Updating..")
            # Save new config
            self.netdata.save_config()

            if self.netdata.start():
                Log.log("Updater: Netdata update complete")
            else:
                Log.log("Updater: Netdata already correct version")

    def update_mc(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log("Updater: Checking for MinIO Client updates")
            # Update payload
            srv = self.payload['miniomc'] 

            # Local info
            loc = self.minio.mc_data

            # Modify if changed
            changed = False
            if srv['repo'] != loc['repo']:
                Log.log(f"Updater: MC repo: {loc['repo']} -> {srv['repo']}")
                loc['repo'] = srv['repo']
                changed = True

            if srv['tag'] != loc['mc_version']:
                Log.log(f"Updater: MC tag: {loc['wireguard_version']} -> {srv['tag']}")
                loc['mc_version'] = srv['tag']
                changed = True

            sha = f"{self.arch}_sha256"
            if srv[sha] != loc[sha]:
                Log.log(f"Updater: MC {sha}: {loc[sha]} -> {srv[sha]}")
                loc[sha] = srv[sha]
                changed = True

            if changed:
                Log.log("Updater: MinIO Client update detected. Updating..")
                # Save new config
                self.minio.save_config()
                if self.minio.start_mc():
                    Log.log("Updater: MinIO Client update complete")
            else:
                Log.log("Updater: MinIO Client already on correct version")


    def update_minio(self):
        if self.config['wgOn'] and self.config['wgRegistered']:
            Log.log("Updater: Checking for MinIO updates")
            copied = self.urbit._urbits
            for p in list(copied):
                # Update payload
                srv = self.payload['minio'] 

                # Local info
                loc = self.urbit._urbits[p]

                name = f"minio_{p}"
                Log.log(f"{name}: Checking for MinIO update")

                # Modify if changed
                changed = False
                if srv['repo'] != loc['minio_repo']:
                    Log.log(f"{name}: MinIO repo: {loc['minio_repo']} -> {srv['repo']}")
                    loc['minio_repo'] = srv['repo']
                    changed = True

                if srv['tag'] != loc['minio_version']:
                    Log.log(f"{name}: MinIO tag: {loc['minio_version']} -> {srv['tag']}")
                    loc['minio_version'] = srv['tag']
                    changed = True

                sha = f"{self.arch}_sha256"
                loc_sha = f"minio_{sha}"
                if srv[sha] != loc[loc_sha]:
                    Log.log(f"{name}: MinIO {sha}: {loc[loc_sha]} -> {srv[sha]}")
                    loc[loc_sha] = srv[sha]
                    changed = True

                if changed:
                    self.urbit.save_config(p)
                    Log.log(f"{name}: MinIO update detected. Updating..")
                    if self.minio.minio_docker.remove_container(name):
                        if self.minio.start_minio(name, self.urbit._urbits[p]):
                            Log.log(f"{name}: MinIO update complete")
                else:
                    Log.log(f"{name}: MinIO already on correct version")


    def update_urbit(self):
        Log.log("Updater: Checking for Urbit updates")
        copied = self.urbit._urbits
        for p in list(copied):
            # Update payload
            srv = self.payload['vere'] 

            # Local info
            loc = self.urbit._urbits[p]

            Log.log(f"{p}: Checking for Urbit update")

            # Modify if changed
            changed = False
            if srv['repo'] != loc['urbit_repo']:
                Log.log(f"{p}: Urbit repo: {loc['urbit_repo']} -> {srv['repo']}")
                loc['urbit_repo'] = srv['repo']
                changed = True

            if srv['tag'] != loc['urbit_version']:
                Log.log(f"{p}: Urbit tag: {loc['urbit_version']} -> {srv['tag']}")
                loc['urbit_version'] = srv['tag']
                changed = True

            sha = f"{self.arch}_sha256"
            loc_sha = f"urbit_{sha}"
            if srv[sha] != loc[loc_sha]:
                Log.log(f"{p}: Urbit {sha}: {loc[loc_sha]} -> {srv[sha]}")
                loc[loc_sha] = srv[sha]
                changed = True

            if changed:
                dupe = self.urbit._urbits[p]
                self.urbit.save_config(p, dupe)
                Log.log(f"{p}: Urbit update detected. Updating..")
                if self.urbit.urb_docker.remove_container(p):
                    if self.urbit.start(p,skip=True) == "succeeded":
                        Log.log(f"{p}: Urbit update complete")
            else:
                Log.log(f"{p}: Urbit already on correct version")
