import os
import asyncio
from time import sleep
from threading import Thread

from groundseg.netdata import Netdata
from groundseg.setup import Setup
from groundseg.startram import StarTramAPI
from groundseg.uploader import Uploader
from groundseg.linux import UpdateLinux
from groundseg.wireguard import Wireguard
from groundseg.minio import MinIO
from groundseg.urbit import Urbit
from groundseg.bug_report import BugReport

from lib.wifi import toggle_wifi, wifi_connect

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
            self.urbit = Urbit(self, self.cfg, self.wireguard, self.minio)

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

    async def process(self, broadcaster, websocket, auth_status, setup, payload):
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

                    if action == "wifi-toggle":
                        res = toggle_wifi()
                        self.cfg.set_wifi_status(res == "on")
                        return

                    if action == "wifi-connect":
                        ssid = payload.get('ssid')
                        password = payload.get('password')
                        wifi_connect(ssid,password)
                        return

                if req_type == "startram":
                    if action == "toggle":
                        Thread(target=self.startram_toggle,args=(broadcaster,)).start()
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
                        Thread(target=self.startram_register,args=(key,region,broadcaster)).start()
                        return

                if req_type == "urbit":
                    patp = payload.get('patp')
                    if action == "register-service-again":
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

                    if action == "toggle-devmode":
                        self.urbit.toggle_devmode(patp)
                        return

                    if action == "toggle-autoboot":
                        self.urbit.toggle_autoboot(patp)
                        return

                    if action == "toggle-network":
                        self.urbit.toggle_network(patp)
                        return

                    if action == "toggle-power":
                        Thread(target=self.urbit.toggle_power, args=(patp,broadcaster)).start()
                        return

                    if action == "delete-ship":
                        Thread(target=self.urbit.delete, args=(patp,self.startram,broadcaster)).start()
                        return

                if req_type == "support":
                    if action == "bug-report":
                        contact = payload.get('contact')
                        description = payload.get('description')
                        ships = payload.get('ships')
                        BugReport.submit_report(
                                contact,
                                description,
                                ships,
                                self.cfg.base,
                                self.cfg.system.get('wgRegistered')
                                )
                        return

            # Execute request
            print(f"groundseg process temporary: auth_status {auth_status}, payload {payload}")
        except Exception as e:
            print(f"groundseg:process: {e}")

    def startram_toggle(self,broadcaster):
        broadcaster.startram.set_transition("toggle","loading")
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
        broadcaster.startram.clear_transition("toggle")

    def startram_register(self,key,region,broadcaster):
        broadcaster.startram.set_transition("register","loading")
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
                            broadcaster.startram.set_transition("register","success")
                            sleep(3)
                            broadcaster.startram.set_transition("register","done")
                            sleep(3)
        broadcaster.startram.clear_transition("register")

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

    def vere_version_info(self):
        print(f"groundseg:vere_version Thread started")
        while True:
            if self.ready:
                try:
                    urbits = self.urbit._urbits.keys()
                    for patp in urbits:
                        try:
                            res = self.urbit.urb_docker.exec(patp, 'urbit --version').output.decode("utf-8").strip().split("\n")[0]
                            self.urbit.set_vere_version(patp, str(res))
                        except:# Exception as e:
                            #print(f"groundseg:vere_version:{patp} {e}")
                            pass
                except Exception as e:
                    print(f"groundseg:vere_version {e}")
                sleep(30)
            else:
                sleep(2)

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
