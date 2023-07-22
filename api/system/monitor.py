import psutil
import shutil
import asyncio

class SysMonitor:
    def __init__(self, cfg, dev):
        super().__init__()
        self.cfg = cfg
        self.dev = dev

    # RAM info
    async def ram(self):
        print("system:monitor:ram: monitor thread started")
        error_time = 15
        while True:
            try:
                self.cfg._ram = [psutil.virtual_memory().total, psutil.virtual_memory().used]
                await asyncio.sleep(1)
                error_time = 15
            except Exception as e:
                self.cfg._ram = 0.0
                print(f"system:monitor:ram: info error: {e}")
                print(f"system:monitor:ram: Checking RAM info again in {error_time} seconds")
                await asyncio.sleep(error_time)
                error_time = error_time * 2

    # CPU info
    async def cpu(self):
        print("system:monitor:cpu CPU monitor thread started")
        error_time = 15
        while True:
            try:
                #self.cfg._cpu = psutil.cpu_percent(1)
                self.cfg._cpu = psutil.cpu_percent(1)
                await asyncio.sleep(1)
                error_time = 15
            except Exception as e:
                self.cfg._cpu = 0.0
                print(f"system:monitor:cpu: CPU info error: {e}")
                print(f"system:monitor:cpu: Checking CPU info again in {error_time} seconds")
                await asyncio.sleep(error_time)
                error_time = error_time * 2

    # CPU Temp info
    async def temp(self):
        print("system:monitor:temp Core temperature monitor thread started")
        error_time = 15
        while True:
            try:
                self.cfg._core_temp = psutil.sensors_temperatures()['coretemp'][0].current
                await asyncio.sleep(1)
                error_time = 15
            except Exception as e:
                self.cfg._core_temp = 0.0
                print(f"system:monitor:temp: Core temperature info error: {e}")
                print(f"system:monitor:temp: Checking core temperature again in {error_time} seconds")
                await asyncio.sleep(error_time)
                error_time = error_time * 2

    # Disk info
    async def disk(self):
        print("system:monitor:disk: Disk monitor thread started")
        error_time = 15
        while True:
            try:
                self.cfg._disk = shutil.disk_usage("/")
                await asyncio.sleep(1)
                error_time = 15
            except Exception as e:
                self.cfg._disk = [0,0,0]
                print(f"system:monitor:disk: Disk info error: {e}")
                print(f"system:monitor:disk: Checking disk info again in {error_time} seconds")
                await asyncio.sleep(error_time)
                error_time = error_time * 2
