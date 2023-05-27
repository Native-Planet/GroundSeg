# Python
import time

# Modules
from cryptography.hazmat.primitives.asymmetric import rsa

# GroundSeg modules
from log import Log


class KeyGen:
    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.keys = config.login_keys
        self.make_keys()

    # Main loop
    def generator_loop(self):
        Log.log("KeyGen: Generator loop started")
        count = 0
        while True:
            try:
                if count < 4:
                    self.wipe_old_keys()
                    count += 1
                else:
                    self.move_keys()
                    self.make_keys()
                    count = 0
            except Exception as e:
                Log.log(f"KeyGen: {e}")

            time.sleep(60)

    # Generate new keys
    def make_keys(self):
        self.keys['cur']['priv'] = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        self.keys['cur']['pub'] = self.keys['cur']['priv'].public_key()

    # Move current keys to old
    def move_keys(self):
        self.keys['old']['pub'] = self.keys['cur']['pub']
        self.keys['old']['priv'] = self.keys['cur']['priv']

    # Wipe old keys
    def wipe_old_keys(self):
        self.keys['old']['pub'] = ""
        self.keys['old']['priv'] = ""
