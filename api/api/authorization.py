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
    def check_token(self,token,websocket):
        if not token:
            # No token was provided, create
            auth_status = False
            token = self.create_token(websocket)
        else:
            # Token was provided verify
            valid, auth_status = self.verify_token(token)
            token = None
            if not valid:
                # Invalid token provided, create
                print("auth:check_token invalid token")
                token = self.create_token(websocket)
        return auth_status, token

    def create_token(self,websocket=None):
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
            "authorized":False,
            "created":now
            }
        # Keyfile location
        k = self.cfg.system.get('keyFile')
        # Encrypted token
        text = self.keyfile_encrypt(contents,k)
        # Update sessions
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

    '''
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
'''
