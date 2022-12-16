from datetime import datetime
import sys

class Log:

    # Log to file
    def log_groundseg(text):
        print(text, file=sys.stderr)
        try:
            with open("/opt/nativeplanet/groundseg/groundseg.log", "a") as log:
                log.write(f"{datetime.now()} {text}\n")
                log.close()
        except:
            pass
