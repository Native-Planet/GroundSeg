import string
import secrets
import hashlib
from datetime import datetime, timedelta

class Auth:
    def __init__(self,cfg):
        self.cfg = cfg

    def check_token(self,token,websocket):
        if not token:
            auth_status = False
            token = self.create_token(websocket)
        else:
            auth_status = self.verify_token(token)
            token = None
        return auth_status, token

    def create_token(self,websocket=None):
        # Default for lick
        ip = '127.0.0.1'
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
        #k = self.config['keyFile']
        # Encrypted token
        #text = self.keyfile_encrypt(contents,k)
        # Update sessions
        '''
        self.config['sessions']['unauthorized'][id] = {
            "hash": self.hash_string(text),
            "created": now
            }
        '''
        # Save config
        self.cfg.save_config()
        # Return token
        text = "placeholder" # temp
        return {"id": id,"token":text}

    def verify_token(self, token):
        return False

    # Randomized string of n length
    def new_secret_string(self,length):
        return ''.join(secrets.choice(
            string.ascii_uppercase + 
            string.ascii_lowercase + 
            string.digits) for i in range(length))
