from datetime import datetime
import sys
import os

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
