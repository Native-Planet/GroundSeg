# Python
import os

# Modules
import nmcli

# GroundSeg modules
from utils import Utils
from log import Log

class SysPost:
    def handle_session(data, config, sid):
        if data['action'] == 'logout':
            config.config['sessions'] = [i for i in config.config['sessions'] if i != sid]
            config.save_config()
            Log.log(f"Login: Logging out off current session: {sid}")
            return 200

        if data['action'] == 'logout-all':
            config.config['sessions'] = []
            config.save_config()
            Log.log("Login: Logging out off all sessions")
            return 200

        if data['action'] == 'change-pass':
            if config.change_password(data):
                return 200

        return 400

    def handle_power(data):
        try:
            if data['action'] == 'shutdown':
                Log.log("Power: Shutdown requested")
                os.system("shutdown -h now")
                return 200

            if data['action'] == 'restart':
                Log.log("Power: Restart requested")
                os.system("reboot")
                return 200

        except Exception as e:
            Log.log(f"Power: Failed: {e}")

        return 400

    def handle_binary(data):
        if data['action'] == 'restart':
            Log.log("GroundSeg: Binary restart requested")
            try:
                os.system("systemctl restart groundseg")
                return 200
            except Exception:
                Log.log("GroundSeg: Binary restart failed: {e}")

        return 400

    def handle_network(data, config):
        if data['action'] == 'toggle':
            try:
                if nmcli.radio.wifi():
                    nmcli.radio.wifi_off()
                    return 200
                else:
                    nmcli.radio.wifi_on()
                    return 200

            except Exception as e:
                Log.log(f"System: Can't toggle ethernet: {e}")

        if data['action'] == 'networks':
            if config.device_mode == "vm":
                return []
            return Utils.list_wifi_ssids()

        if data['action'] == 'connect':
            if Utils.wifi_connect(data['network'], data['password']):
                return 200

        return 400

    def handle_updater(data, config):
        if data['action'] == 'toggle':
            try:
                mode = config.config['updateMode']
                if mode == 'auto':
                    config.config['updateMode'] = 'off'
                else:
                    config.config['updateMode'] = 'auto'
                Log.log(f"Updater: Update mode changed. {mode} -> {config.config['updateMode']}")
                config.save_config()
                return 200
            except Exception as e:
                Log.log(f"Updater: Failed to change Update mode: {e}")

            return 400
