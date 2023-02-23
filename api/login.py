#Python
import string
import secrets
import hashlib

# Modules
from flask import make_response, jsonify

# GroundSeg modules
from log import Log

class Login:
    def handle_login(data, config):
        Log.log("Login: Login requested")
        if 'password' in data:
            encoded_str = (config.config['salt'] + data['password']).encode('utf-8')
            this_hash = hashlib.sha512(encoded_str).hexdigest()

            if this_hash == config.config['pwHash']:
                Log.log("Login: Password is correct!")
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

    def failed():
        Log.log("Login: Rejecting login request")
        return make_response(jsonify(400))
