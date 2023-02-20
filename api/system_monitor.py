import psutil
import shutil

from time import sleep
from log import Log

class SysMonitor:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config

    def sys_monitor(self):
        Log.log("System monitor thread started")
        error = False
        while not self.config_object.device_mode == "vm":
            if error:
                Log.log("System monitor error, 15 second timeout")
                sleep(15)
                error = False

            # RAM info
            try:
                self.config_object._ram = psutil.virtual_memory().percent
            except Exception as e:
                self.config_object._ram = 0.0
                Log.log(f"RAM info error: {e}")
                error = True

            # CPU info
            try:
                self.config_object._cpu = psutil.cpu_percent(1)
            except Exception as e:
                self.config_object._cpu = 0.0
                Log.log(f"CPU info error: {e}")
                error = True

            # CPU Temp info
            try:
                self.config_object._core_temp = psutil.sensors_temperatures()['coretemp'][0].current
            except Exception as e:
                self.config_object._core_temp = 0.0
                Log.log(f"Core Temp info error: {e}")
                error = True

            # Disk info
            try:
                self.config_object._disk = shutil.disk_usage("/")
            except Exception as e:
                self.config_object._disk = [0,0,0]
                Log.log(f"Disk info error: {e}")
                error = True
