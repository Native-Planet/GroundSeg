# Python
import os
import json
import zipfile
import requests
import subprocess
from datetime import datetime, timedelta

# GroundSeg modules
from log import Log

class BugReport:
    def submit_report(data, base_path, wg_reg):
        Log.log("Bug: Attempting to send bug report")
        try:
            # Current date
            now = datetime.now()

            # current logfile
            current_logfile = f"{now.strftime('%Y-%m')}.log"

            # Previous logfile
            if now.month == 1:
                prev = datetime(now.year - 1, 12, now.day)
            else:
                prev = now - timedelta(days=now.day)
            prev_logfile = f"{prev.strftime('%Y-%m')}.log"

            # Make report
            report = now.strftime('%Y-%m-%d-%H-%M-%S')
            os.system(f"mkdir -p {base_path}/bug-reports/{report}")
            with open(f"{base_path}/bug-reports/{report}/details.txt", "w") as f:
                Log.log(f"Bug: Saving bug report {report} locally")
                f.write(f"Contact:\n{data['person']}\nDetails:\n{data['message']}")
                f.close()

            # Create zipfile
            bug_file = zipfile.ZipFile(
                    f"{base_path}/bug-reports/{report}/{report}.zip", 'w', zipfile.ZIP_DEFLATED
                    )

            # Pier logs
            try:
                for p in data['logs']:
                    try:
                        bug_file.writestr(f'{p}.log', subprocess.check_output(['docker', 'logs', p]).decode('utf-8'))
                    except Exception as e:
                        Log.log(f"Bug: Failed to get {p} logs: {e}")
                        bug_file.writestr(f'failed_{p}.log',e)
            except Exception as e:
                Log.log(f"Bug: Failed to get pier logs: {e}")
                bug_file.writestr('failed_pier_logs',e)

            # Load configs
            try:
                cfgs = {}
                for j in [c for c in os.listdir(f"{base_path}/settings") if c.endswith(".json")]:
                    try:
                        with open(f"{base_path}/settings/{j}") as f:
                            cfgs[j] = json.load(f)

                        if j == "system.json":
                            cfgs[j].pop("sessions")
                            cfgs[j].pop("privkey")
                            cfgs[j].pop("salt")
                            cfgs[j].pop("pwHash")

                        bug_file.writestr(j, json.dumps(cfgs[j], indent = 4))
                    except Exception as e:
                        Log.log(f"Bug: Failed to load {j}: {e}")
                        bug_file.writestr(f"failed_{j}", e)

            except Exception as e:
                Log.log(f"Bug: Failed to load configs: {e}")
                bug_file.writestr("cfgs_failed", e)

            # Load pier configs
            try:
                pcfgs = {}
                for j in [c for c in os.listdir(f"{base_path}/settings/pier") if c.endswith(".json")]:
                    try:
                        with open(f"{base_path}/settings/pier/{j}") as f:
                            pcfgs[j] = json.load(f)

                        pcfgs[j].pop("minio_password")

                        bug_file.writestr(j, json.dumps(pcfgs[j], indent = 4))
                    except Exception as e:
                        Log.log(f"Bug: Failed to load {j}: {e}")
                        bug_file.writestr(f"failed_{j}", e)

            except Exception as e:
                Log.log(f"Bug: Failed to load pier configs: {e}")
                bug_file.writestr("pier_cfgs_failed", e)

            # wireguard config
            if wg_reg:
                try:
                    bug_file.writestr('wireguard.log', subprocess.check_output(['docker', 'logs', 'wireguard']))
                    bug_file.writestr('wg_show.txt', subprocess.check_output(
                        ['docker', 'exec', 'wireguard', 'wg', 'show']
                        ))
                except:
                    Log.log("Bug: Unable to get wireguard logs")

            # docker ps -a
            bug_file.writestr('docker.txt', subprocess.check_output(['docker', 'ps', '-a']).decode('utf-8'))

            # previous log
            try:
                bug_file.write(f"{base_path}/logs/{prev_logfile}", arcname=prev_logfile)
            except:
                pass


            # current log
            bug_file.write(f"{base_path}/logs/{current_logfile}", arcname=current_logfile)

            # save zipfile
            bug_file.close()

            # send to endpoint
            bug_endpoint = "https://bugs.groundseg.app"


            uploaded_file = open(f"{base_path}/bug-reports/{report}/{report}.zip", 'rb')

            form_data = {"contact": data['person'], "string": data['message']}
            form_file = {"zip_file": (f"{report}.zip", uploaded_file)}

            r = requests.post(bug_endpoint, data=form_data, files=form_file)
            Log.log(f"Bug: Report sent on {report}")

            return r.status_code

        except Exception as e:
            Log.log(f"Bug: Failed to send report: {e}")

        return 400
