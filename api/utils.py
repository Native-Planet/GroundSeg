from datetime import datetime
import sys
import os
import nmcli

class Log:

    # Log to file
    def log_groundseg(text):
        print(text, file=sys.stderr)
        try:
            # make directory if doesn't exist
            os.system("mkdir -p /opt/nativeplanet/groundseg/logs")

            # current log file
            current_logfile = f"{datetime.now().strftime('%Y-%m')}.log"

            # move legacy logfile to new directory
            if os.path.isfile(f"/opt/nativeplanet/groundseg/groundseg.log"):
                os.system(f"mv /opt/nativeplanet/groundseg/groundseg.log /opt/nativeplanet/groundseg/logs/{current_logfile}")

            # write to logfile
            with open(f"/opt/nativeplanet/groundseg/logs/{current_logfile}", "a") as log:
                log.write(f"{datetime.now()} {text}\n")
                log.close()
        except:
            pass

class Network:

    def list_wifi_ssids():
        return [x.ssid for x in nmcli.device.wifi() if len(x.ssid) > 0]

    def wifi_connect(ssid, pwd):
        try:
            nmcli.device.wifi_connect(ssid, pwd)
            return "success"
        except Exception as e:
            return f"failed: {e}"
