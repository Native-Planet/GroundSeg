import os
import json
import string
import secrets
import hashlib
from datetime import datetime, timedelta

from cryptography.fernet import Fernet

class Auth:
    def __init__(self, state):
        self.state = state
        self.config_object = self.state['config']
        self.config = self.config_object.config

        self.authorized = self.state['clients']['authorized']
        self.unauthorized = self.state['clients']['unauthorized']

    def verify_session(self, data, websocket):
        token = data.get('token')
        cat = data['payload']['category']
        token_object = None

        try:
            if token == None:
                raise Exception("no token")

            i = token['id']
            t = token['token']

            if self.check_token_hash(i,t):
                d = self.keyfile_decrypt(t,self.config['keyFile'])
                if self.check_token_content(websocket,d):
                    if d.get('authorized'):
                        self.authorized[websocket] = token
                        try:
                            self.unauthorized.pop(websocket)
                        except:
                            pass
                    else:
                        self.unauthorized[websocket] = token
                        try:
                            self.authorized.pop(websocket)
                        except:
                            pass

                    status_code = 0
                    msg = "RECEIVED"

                else:
                    raise Exception("incorrect contents")
            else:
                raise Exception("hash mismatch")
        except Exception as e:
            print(f"auth:verify_session {e}")

            if cat == "token":
                token_object = self.create_token(data,websocket)
                status_code = 2
                msg = "NEW_TOKEN"
            else:
                status_code = 1
                msg = "UNAUTHORIZED"

        return status_code, msg, token_object

    # Check Session Hash
    def check_token_hash(self, id, token):
        s = self.config['sessions']
        a = s['authorized']
        u = s['unauthorized']
        h = self.hash_string(token)

        if id in a:
            if h == a[id]['hash']:
                return True
        if id in u:
            if h == u[id]['hash']:
                return True

        return False

    # hash string
    def hash_string(self,s):
        # Create a new SHA256 hash object
        hash_object = hashlib.sha256()
        # Update the hash object with the bytes of the string
        hash_object.update(s.encode('utf-8'))
        # Get the hexadecimal representation of the hash
        hex_dig = hash_object.hexdigest()
        return hex_dig

    # Create a random string of characters
    def new_secret_string(self, length):
        secret = ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(length))
        return secret


    #
    #   AES Keyfile
    #

    # Encrypt with keyfile
    def keyfile_encrypt(self, contents, key):
        if not os.path.isfile(key):
            print(f"auth:keyfile_encrypt {key} does not exist. Creating")
            k = Fernet.generate_key()
            with open(key,"wb") as f:
                f.write(k)
                f.close()
        else:
            with open(key,"rb") as f:
                k = f.read()
        cipher_suite = Fernet(k)
        data = json.dumps(contents).encode('utf-8')
        text = cipher_suite.encrypt(data)
        return text.decode('utf-8')

    # Decrypt with keyfile
    def keyfile_decrypt(self, text, key):
        if not os.path.isfile(key):
            print(f"auth:keyfile_decrypt {key} does not exist. Returning None")
            return None
        else:
            with open(key,"rb") as f:
                k = f.read()
        cipher_suite = Fernet(k)
        data = cipher_suite.decrypt(text.encode('utf-8'))
        return json.loads(data)

    # Check if content of token is valid
    def check_token_content(self, websocket, token):
        if websocket.remote_address[0] != token['ip']:
            return False

        if websocket.request_headers.get('User-Agent') != token['user_agent']:
            return False

        expire = datetime.strptime(token['created'], "%Y-%m-%d_%H:%M:%S") + timedelta(days=30)
        now = datetime.now()
        if expire <= now:
            return False

        return True

    def create_token(self, data, websocket):
        ip = websocket.remote_address[0]
        user_agent = websocket.request_headers.get('User-Agent')
        cat = data['payload']['category']
        if cat == "token":
            # create token
            id = self.new_secret_string(32)
            secret = self.new_secret_string(128)
            padding = self.new_secret_string(32)
            now = datetime.now().strftime("%Y-%m-%d_%H:%M:%S")
            contents = {
                    "id":id,
                    "ip":ip,
                    "user_agent":user_agent,
                    "secret":secret,
                    "padding":padding,
                    "authorized":False,
                    "created":now
                    }
            k = self.config['keyFile']
            text = self.keyfile_encrypt(contents,k)
            self.config['sessions']['unauthorized'][id] = {
                    "hash": self.hash_string(text),
                    "created": now
                    }
            self.config_object.save_config()
            return {
                    "token": {
                        "id":id,
                        "token":text
                        }
                    }

    def authorize_token(self, token):
        # decrypt
        k = self.config['keyFile']
        contents = self.keyfile_decrypt(token['token'],k)

        # authorize token
        contents['authorized'] = True
        token['token'] = self.keyfile_encrypt(contents,k)

        # get current token
        id = contents['id']
        unauth = self.config['sessions']['unauthorized'][id]

        # modify the token hash
        unauth['hash'] = self.hash_string(token['token'])

        # move token to authorized
        self.config['sessions']['authorized'][id] = unauth

        # remove token from unauthorized
        self.config['sessions']['unauthorized'].pop(id)

        # save changes
        self.config_object.save_config()

        return {"token":token}

    def handle_login(self, action, websocket, status_code, msg):
        if websocket in self.unauthorized:
            try:
                pwd = action['payload']['action']['password']
            except:
                pwd = ""

            # check if password is correct
            from config.utils import Utils
            if Utils.compare_password(self.config['salt'], pwd, self.config['pwHash']):
            #if self.check_password(pwd):
                token = self.authorize_token(action.get('token'))
                status_code = 3
                msg = "AUTHORIZED"
                self.authorized[websocket] = token
                try:
                    self.unauthorized.pop(websocket)
                except:
                    pass
            else:
                token = action.get('token')
                status_code = 1
                msg = "AUTH_FAILED"

        return status_code, msg, token
