#Python
import string
import secrets
from datetime import datetime, timedelta

# Modules
from flask import make_response, jsonify

# GroundSeg modules
from log import Log
from utils import Utils

class Login:
    def handle_login(data, config):
        Log.log("Login: Login requested")
        if 'password' in data:
            Log.log("Login: Attempting with current key")
            decrypted = Utils.decrypt_password(config.login_keys['cur']['priv'], data['password'])
            if Utils.compare_password(config.config['salt'], decrypted, config.config['pwHash']):
                Log.log("Login: Password is correct!")
                config.login_status = {"locked": False, "end": datetime(1,1,1,0,0), "attempts": 0}
                return True

            Log.log("Login: Attempting with previous key")
            decrypted = Utils.decrypt_password(config.login_keys['old']['priv'], data['password'])
            if Utils.compare_password(config.config['salt'], decrypted, config.config['pwHash']):
                Log.log("Login: Password is correct!")
                config.login_status = {"locked": False, "end": datetime(1,1,1,0,0), "attempts": 0}
                return True


        Log.log("Login: Password incorrect")
        return False

    def make_cookie(config):
        secret = ''.join(secrets.choice(
            string.ascii_uppercase +
            string.ascii_lowercase +
            string.digits) for i in range(64))

        config.config['sessions'].append(secret)
        config.save_config()
        Log.log("Login: Created new Session ID")

        res = make_response(jsonify(200))
        res.set_cookie('sessionid', secret)
        Log.log(f"Login: Active Sessions {len(config.config['sessions'])}")

        return res

    def failed(config, unlocked):
        if unlocked:
            attempt = config.login_status['attempts']
            attempt += 1
            if attempt == 3:
                config.login_status['end'] = datetime.now() + timedelta(seconds=60)
                config.login_status['locked'] = True

            if attempt == 4:
                config.login_status['end'] = datetime.now() + timedelta(seconds=300)
                config.login_status['locked'] = True

            if attempt == 5:
                config.login_status['end'] = datetime.now() + timedelta(seconds=1800)
                config.login_status['locked'] = True

            if attempt == 6:
                config.login_status['end'] = datetime.now() + timedelta(seconds=3600)
                config.login_status['locked'] = True

            if attempt == 7:
                config.login_status['end'] = datetime.now() + timedelta(seconds=(3600 * 6))
                config.login_status['locked'] = True

            if attempt == 8:
                config.login_status['end'] = datetime.now() + timedelta(seconds=(3600 * 12))
                config.login_status['locked'] = True

            if attempt == 9:
                config.login_status['end'] = datetime.now() + timedelta(seconds=(3600 * 24))
                config.login_status['locked'] = True

            if attempt >= 10:
                config.login_status['end'] = datetime.now() + timedelta(seconds=(3600 * 72))
                config.login_status['locked'] = True

            config.login_status['attempts'] = attempt
            Log.log(f"Login: Rejecting login request: Attempt {attempt}")
        return make_response(jsonify(400))
