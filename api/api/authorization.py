import os
import json
import string
import secrets
import hashlib
from datetime import datetime, timedelta

from cryptography.fernet import Fernet

class Auth:
    def __init__(self,cfg):
        self.cfg = cfg

    # Check validity of token
    def check_token(self,token,websocket,setup=False):
        if not token:
            # No token was provided, create
            auth_status = False
            # Special case for setup
            if setup:
                auth_status = True
            token = self.create_token(websocket,setup)
        else:
            # Token was provided verify
            valid, auth_status = self.verify_token(token)
            if not valid:
                # Invalid token provided, create
                print("auth:check_token invalid token")
                token = self.create_token(websocket,setup)
        return auth_status, token

    def create_token(self,websocket=None,setup=False):
        print(f"auth:create_token creating new session token")
        # Default for lick
        ip = 'localhost'
        user_agent = 'lick'
        # Set websocket information
        if websocket:
            ip = websocket.remote_address[0]
            user_agent = websocket.request_headers.get('User-Agent')

        # Generate random string for id
        id = self.new_secret_string(32)
        # Generate randomstring for secret
        secret = self.new_secret_string(128)
        # Added padding for no reason except for fun
        padding = self.new_secret_string(32)
        # Set time of creation
        now = datetime.now().strftime("%Y-%m-%d_%H:%M:%S")
        # Unencrypted contents of token
        contents = {
            "id":id,
            "ip":ip,
            "user_agent":user_agent,
            "secret":secret,
            "padding":padding,
            "authorized":setup,
            "created":now
            }
        # Keyfile location
        k = self.cfg.system.get('keyFile')
        # Encrypted token
        text = self.keyfile_encrypt(contents,k)
        # Update sessions
        if setup:
            self.cfg.system['sessions']['authorized'][id] = {
                "hash": self.hash_string(text),
                "created": now
                }
        else:
            self.cfg.system['sessions']['unauthorized'][id] = {
                "hash": self.hash_string(text),
                "created": now
                }
        # Save config
        self.cfg.save_config()
        # Return token
        return {"id": id,"token":text}

    def verify_token(self, token):
        valid = False
        auth_status = False
        # Get sessions from config
        s = self.cfg.system.get('sessions')
        a = s['authorized']
        u = s['unauthorized']
        # Get id and token cypher text 
        id = token.get('id')
        cypher = token.get('token')
        # Make sure cypher and id exists
        if (not cypher) or (not id):
            return False
        # Hash provided token
        h = self.hash_string(cypher)
        # Check if token hash exists in authorized
        if id in a:
            if h == a[id]['hash']:
                valid = True
                auth_status = True
        # Check if token hash exists in unauthorized
        if id in u:
            if h == u[id]['hash']:
                valid = True
                auth_status = False
        return valid, auth_status

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

    def handle_logout(self,token,websocket,action=None):
        id = token.get('id')
        # remove from cfg
        self.cfg.system['sessions']['authorized'].pop(id)
        # save config
        self.cfg.save_config()
        remove_from = "authorized"
        auth_status = False
        if action == "everywhere":
            auth_status = True

        return remove_from, auth_status, token

    def handle_login(self,token,password,websocket):
        # check if password is correct
        remove_from = "none"
        auth_status = self.cfg.check_password(password)
        if auth_status:
            k = self.cfg.system.get('keyFile')
            # decrypt the token provided by the user
            x = self.keyfile_decrypt(token.get('token'),k)
            # check if content matches the session
            valid = self.check_token_content(websocket, x)
            if valid:
                # modify token content
                token = self.authorize_token(x,token.get('id'))
                remove_from = "unauthorized"
            else:
                auth_status = False

        return remove_from, auth_status, token

    # Randomized string of n length
    def new_secret_string(self,length):
        return ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(length))

    # hash string
    def hash_string(self,s):
        # Create a new SHA256 hash object
        hash_object = hashlib.sha256()
        # Update the hash object with the bytes of the string
        hash_object.update(s.encode('utf-8'))
        # Get the hexadecimal representation of the hash
        hex_dig = hash_object.hexdigest()
        return hex_dig

    # Set token authorization to True
    def authorize_token(self, decrypted, id):
        # authorize token
        decrypted['authorized'] = True
        encrypted = self.keyfile_encrypt(decrypted,self.cfg.system.get('keyFile'))
        token = {"id":id,"token":encrypted}

        # modify the token hash
        unauth = self.cfg.system['sessions']['unauthorized'][id]
        unauth['hash'] = self.hash_string(token['token'])

        # move token to authorized
        self.cfg.system['sessions']['authorized'][id] = unauth

        # remove token from unauthorized
        self.cfg.system['sessions']['unauthorized'].pop(id)

        # save changes
        self.cfg.save_config()

        return token


    # Encrypt with keyfile
    def keyfile_encrypt(self, contents, key):
        # Check if keyfile exists
        if not os.path.isfile(key):
            print(f"auth:keyfile_encrypt {key} does not exist. Creating")
            # Generate new key
            k = Fernet.generate_key()
            # Write to file
            with open(key,"wb") as f:
                f.write(k)
                f.close()
        else:
            # Open the keyfile
            with open(key,"rb") as f:
                # Read the key
                k = f.read()
        # Initialize cipher suite
        cipher_suite = Fernet(k)
        # Load and encrypt text
        data = json.dumps(contents).encode('utf-8')
        text = cipher_suite.encrypt(data)
        # Return decoded text
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
