import psutil
import shutil

from time import sleep
from log import Log

class SysMonitor:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.mode = config.device_mode

    # RAM info
    def ram_monitor(self):
        Log.log("Monitor: RAM monitor thread started")
        error_time = 15
        while True:
            try:
                self.config_object._ram = psutil.virtual_memory().percent
                sleep(1)
                error_time = 15
            except Exception as e:
                self.config_object._ram = 0.0
                Log.log(f"Monitor: RAM info error: {e}")
                Log.log(f"Monitor: Checking RAM info again in {error_time} seconds")
                sleep(error_time)
                error_time = error_time * 2

    # CPU info
    def cpu_monitor(self):
        Log.log("Monitor: CPU monitor thread started")
        error_time = 15
        while True:
            try:
                self.config_object._cpu = psutil.cpu_percent(1)
                error_time = 15
            except Exception as e:
                self.config_object._cpu = 0.0
                Log.log(f"Monitor: CPU info error: {e}")
                Log.log(f"Monitor: Checking CPU info again in {error_time} seconds")
                sleep(error_time)
                error_time = error_time * 2

    # CPU Temp info
    def temp_monitor(self):
        Log.log("Monitor: Core temperature monitor thread started")
        error_time = 15
        while True:
            try:
                self.config_object._core_temp = psutil.sensors_temperatures()['coretemp'][0].current
                sleep(1)
                error_time = 15
            except Exception as e:
                self.config_object._core_temp = 0.0
                Log.log(f"Monitor: Core temperature info error: {e}")
                Log.log(f"Monitor: Checking core temperature again in {error_time} seconds")
                sleep(error_time)
                error_time = error_time * 2

    # Disk info
    def disk_monitor(self):
        Log.log("Monitor: Disk monitor thread started")
        error_time = 15
        while True:
            try:
                self.config_object._disk = shutil.disk_usage("/")
                sleep(1)
                error_time = 15
            except Exception as e:
                self.config_object._disk = [0,0,0]
                Log.log(f"Monitor: Disk info error: {e}")
                Log.log(f"Monitor: Checking disk info again in {error_time} seconds")
                sleep(error_time)
                error_time = error_time * 2
