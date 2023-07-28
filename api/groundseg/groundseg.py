import os
import asyncio
from time import sleep

from groundseg.netdata import Netdata
from groundseg.setup import Setup
from groundseg.startram import StarTramAPI
from groundseg.uploader import Uploader
from groundseg.linux import UpdateLinux
from groundseg.wireguard import Wireguard
from groundseg.minio import MinIO
from groundseg.urbit import Urbit

class GroundSeg:
    def __init__(self, cfg, dev):
        self.cfg = cfg
        self.dev = dev
        self.ready = False
        self.active_sessions = {
                "authorized": {},
                "unauthorized": {}
                }

    # The groundseg loader initializes the main app
    async def loader(self):
        # We start by making sure the version info has
        # been retrieved if updateMode is set to auto
        try:
            if self.cfg.system.get('updateMode') == 'auto':
                count = 0
                while not self.cfg.version_server_ready:
                    count += 1
                    # We cancel this check after 10 tries (30 seconds)
                    if count >= 10:
                        print("groundseg:groundseg:loader: Failed after 10 tries")
                        break
                    print("groundseg:groundseg:loader: Waiting for update server...")
                    await asyncio.sleep(3)

            # We load the setup class first
            self.setup = Setup(self,self.cfg)

            # Uploader
            self.uploader = Uploader(self,self.cfg)

            # After that, we load netdata
            self.netdata = Netdata(self.cfg)

            # And then we load wireguard. This is a prerequisite for any startram
            # related functionality.
            self.wireguard = Wireguard(self.cfg)

            # Lastly, we load ship specific classes.
            # MinIO - s3 bucket for individual ships
            # Urbit - Urbit ship. Also handles Lick side permissions (See authorization)
            self.minio = MinIO(self.cfg, self.wireguard)
            self.urbit = Urbit(self.cfg, self.wireguard, self.minio)

            # StarTram API
            self.startram = StarTramAPI(self.cfg,self.wireguard,self.urbit)

            # This will be deprecated eventually in preference of
            # static html/js
            #self.webui = WebUI(self.cfg)

            # Inform the APIs that groundseg is now ready
            print("groundseg:groundseg:loader: Initialization completed")
            self.ready = True
        except Exception as e:
            print(f"groundseg:groundseg:loader error {e}")

    async def process(self, websocket, auth_status, setup, payload):
        try:
            req_type = payload.get('type')
            action = payload.get('action')
            # Check if setup
            if setup:
                if req_type == "setup":
                    if action == "begin":
                        self.setup.begin()
                        return
                    if action == "password":
                        pwd = payload.get('password')
                        if pwd:
                            self.setup.password(pwd)
                        return
                    if action == "skip":
                        self.setup.complete(websocket)
                        return
                    if action == "startram":
                        key = payload.get('key')
                        region = payload.get('region')
                        # register device
                        if self.startram.register_device(key,region):
                            # retrieve latest status from startram
                            if self.startram.retrieve_status():
                                # start wg
                                if self.wireguard.start():
                                # write wg0.conf
                                    if self.wireguard.write_wg_conf():
                                        # set complete
                                        self.setup.complete(websocket)
                        return

            # Check if auth
            if auth_status:
                if req_type == "new_ship":
                    if action == "boot":
                        patp = payload.get('patp')
                        key = payload.get('key')
                        remote = payload.get('remote')
                        res = await self.urbit.create(patp,key,remote)
                        # register services
                        if res and self.cfg.system.get('wgRegistered'):
                            self.startram.create_service(patp, 'urbit')
                            self.startram.create_service(f"s3.{patp}", 'minio')
                        return
                #
                if req_type == "pier_upload":
                    #
                    if action == "free":
                        self.uploader.make_free()
                        return
                    #
                    if action == "metadata":
                        patp = payload.get('patp')
                        size = payload.get('size')
                        secret = payload.get('secret')
                        if patp and size and secret:
                            if self.uploader.open_http(secret):
                                self.uploader.update_metadata(patp,size)
                        return

                if req_type == "password":
                    if action == "modify":
                        old = payload.get('old')
                        pwd = payload.get('password')
                        if self.cfg.check_password(old):
                            self.cfg.create_password(pwd)
                        return

                if req_type == "system":
                    if action == "restart":
                        if self.dev:
                            print("groundseg:process:system:restart: Dev mode, skipping")
                        else:
                            os.system("systemcl restart groundseg")
                        return

                    if action == "update":
                        update = payload.get('update')
                        if update == "linux":
                            UpdateLinux(self.cfg,self.dev).run_update()
                        return

                    if action == "modify-swap":
                        val = int(payload.get('value'))
                        self.cfg.set_swap(val)
                        return

                    if action == "power":
                        command = payload.get('command')
                        if command == "restart":
                            if self.dev:
                                print("groundseg:process:system:power:restart: Dev mode, skipping restart")
                            else:
                                os.system("reboot")
                        if command == "shutdown":
                            if self.dev:
                                print("groundseg:process:system:power:shutdown: Dev mode, skipping shutdown")
                            else:
                                os.system("shutdown -h now")
                        return

                if req_type == "startram":
                    if action == "toggle":
                        # if running
                        if self.wireguard.is_running():
                            for p in self.urbit._urbits.copy():
                                if self.urbit._urbits[p]['network'] == "wireguard":
                                    # swap ship back to local
                                    self.urbit.toggle_network(p)
                                    # turn off minio
                            # turn off wireguard
                            self.wireguard.stop()
                        else:
                            # turn on wireguard
                            self.wireguard.start()
                            # start minio
                        return

                    if action == "restart":
                        payload['action'] = "toggle"
                        ships = []
                        if self.wireguard.is_running():
                            for p in self.urbit._urbits.copy():
                                if self.urbit._urbits[p]['network'] == "wireguard":
                                    ships.append(p)
                            await self.process(websocket, auth_status, setup, payload)
                        await self.process(websocket, auth_status, setup, payload)
                        # select ships that should be in remote
                        for p in ships:
                            self.urbit.toggle_network(p)
                        return

                    if action == "regions":
                        self.startram.get_regions(3)
                        return

                    if action == "endpoint":
                        url = payload.get('endpoint')
                        if self.wireguard.stop():
                            if self.startram.unregister():
                                if self.cfg.set_endpoint(url):
                                    self.cfg.reset_pubkey()
                        return

                    if action == "cancel":
                        key = payload.get('key')
                        reset = payload.get('reset')
                        if reset:
                            if self.startram.unregister():
                                self.cfg.reset_pubkey()
                        self.startram.cancel_subscription(key)
                        return

                    if action == "register":
                        key = payload.get('key')
                        region = payload.get('region')
                        # reset the pubkey
                        if self.cfg.reset_pubkey():
                            # TODO: turn urbits and minios to local
                            # set wgRegistered and wgOn false
                            self.cfg.system['wgRegistered'] = False
                            self.cfg.system['wgOn'] = False
                            self.cfg.save_config()
                            # remove wireguard container
                            if self.wireguard.remove():
                                # register device
                                if self.startram.register_device(key,region):
                                    # retrieve latest status from startram
                                    if self.startram.retrieve_status():
                                        for p in self.cfg.system.get('piers'):
                                            self.startram.create_service(p, 'urbit')
                                            self.startram.create_service(f"s3.{p}", 'minio')
                                        # start wg
                                        if self.wireguard.start():
                                            # write wg0.conf
                                            self.wireguard.write_wg_conf()
                                            self.wireguard.register_broadcast_status = "success"
                                            sleep(3)
                                            self.wireguard.register_broadcast_status = "done"
                                            sleep(3)
                                            self.wireguard.register_broadcast_status = None
                        return

                if req_type == "urbit":
                    if action == "register-service-again":
                        def get_patp(url):
                            return url.split('.')[0]

                        if self.cfg.system.get('wgRegistered'):
                            patp = payload.get('patp')
                            # check if services exist already
                            if self.startram.retrieve_status(3):
                                if patp in self.wireguard.anchor_services:
                                    self.startram.delete_service(patp, 'urbit')
                                    self.startram.delete_service(f"s3.{patp}", 'minio')

                                self.startram.create_service(patp, 'urbit')
                                self.startram.create_service(f"s3.{patp}", 'minio')
                        return

                if req_type == "support":
                    if action == "bug-report":
                        contact = payload.get('contact')
                        description = payload.get('description')
                        ships = payload.get('ships')
                        print("DO SOMETHING")
                        print("BUG REPORT")
                        print(f"CONTACT: {contact}")
                        print(f"DESC: {description}")
                        print(f"SHIPS: {ships}")
                        print("DO SOMETHING")
                        return

            # Execute request
            print(f"groundseg process temporary: auth_status {auth_status}, payload {payload}")
        except Exception as e:
            print(f"groundseg:process: {e}")

    def handle_upload(self,req):
        return self.uploader.handle_chunk(req)

    async def startram(self):
        await self.startram.main_loop()

    # Started as a thread
    def urbit_docker_stats(self):
        while True:
            if self.ready:
                self.urbit.ram()
            sleep(1)

    async def melder(self):
        while True:
            try:
                if self.ready:
                    #print("checking for melds")
                    pass
                else:
                    print("gs not ready")
            except Exception as e:
                print(f"meld loop error {e}")
            await asyncio.sleep(1)

    async def wg_refresher(self):
        while True:
            try:
                if self.ready:
                    #print("checking for wireguard connection health")
                    pass
                else:
                    print("gs not ready")
            except Exception as e:
                print(f"wg_refresher error {e}")
            await asyncio.sleep(1)

    async def docker_updater(self):
        while True:
            try:
                if self.ready:
                    #print("checking for docker updates")
                    pass
                else:
                    print("gs not ready")
            except Exception as e:
                print(f"docker_updater error {e}")
            await asyncio.sleep(1)
