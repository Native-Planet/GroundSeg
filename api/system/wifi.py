import nmcli
import asyncio

from lib.wifi import list_wifi_ssids

class WifiNetwork:
    def __init__(self,cfg):
        self.cfg = cfg

    async def get_wifi_status(self):
        while True:
            try:
                on = nmcli.radio.wifi()
                self.cfg.set_wifi_status(on)
            except Exception as e:
                print(f"system:wifi:get_wifi_status: Can't get wifi status {e}")
                self.cfg.set_wifi_status(False)
            await asyncio.sleep(5)

    async def get_active_wifi(self):
        while True:
            name = None
            if self.cfg._wifi_enabled:
                try:
                    conns = nmcli.connection()
                    for con in conns:
                        if con.conn_type == "wifi":
                            name = con.name
                            break
                except:
                    print("system:wifi:get_active_wifi Can't get WiFi connection status")
            self.cfg.set_active_wifi(name)
            await asyncio.sleep(5)

    async def get_wifi_list(self):
        while True:
            ssids = []
            if self.cfg._wifi_enabled:
                try:
                    ssids = list_wifi_ssids()
                except:
                    pass
            self.cfg.set_wifi_networks(ssids)
            await asyncio.sleep(30)

        '''
        if data['action'] == 'connect':
            if Utils.wifi_connect(data['network'], data['password']):
                return 200

        return 400
        '''
