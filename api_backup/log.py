import os
import sys
import shutil
from datetime import datetime

class Log:

    # Log to file
    def log(text):
        print(text, file=sys.stderr)
        try:
            # make directory if doesn't exist
            os.makedirs("/opt/nativeplanet/groundseg/logs", exist_ok=True)

            # current log file
            current_logfile = f"{datetime.now().strftime('%Y-%m')}.log"

            # move legacy logfile to new directory
            if os.path.isfile("/opt/nativeplanet/groundseg/groundseg.log"):
                shutil.move("/opt/nativeplanet/groundseg/groundseg.log",
                            f"/opt/nativeplanet/groundseg/logs/{current_logfile}")

            # write to logfile
            with open(f"/opt/nativeplanet/groundseg/logs/{current_logfile}", "a") as log:
                log.write(f"{datetime.now()} {text}\n")
                log.close()
        except:
            pass

    # Get GroundSeg logs
    def get_log():
        # current log file
        current_logfile = f"{datetime.now().strftime('%Y-%m')}.log"

        # read log
        with open(f"/opt/nativeplanet/groundseg/logs/{current_logfile}") as f:
            log = f.read()
            f.close()

        return log.split("\n")
