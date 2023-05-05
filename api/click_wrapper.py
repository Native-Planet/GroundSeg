import os
from log import Log

class Click:
    def __init__(self, patp="zod", name=None, urb=None):
        self.patp = patp
        self.urb = urb
        self.name = name

    def run(self, payload={}):
        try:
            # code
            if self.name == "code":
                if self.create_hoon():
                    raw = self.click_exec(
                            self.patp,
                            self.urb.urb_docker.exec,
                            f"{self.name}.hoon",
                            True
                            )
                    self.delete_hoon()
                    return self.filter_code(raw)

            # s3
            if self.name in ["s3","s3-legacy"]:
                if self.create_hoon(payload):
                    raw = self.click_exec(
                            self.patp,
                            self.urb.urb_docker.exec,
                            f"{self.name}.hoon",
                            True
                            )
                    self.delete_hoon()
                return self.filter_success(raw)

        except Exception as e:
            Log.log(f"(WS)Click: run() failed: {e}")
        return None

    def click_exec(self, patp, docker_exec, hoon_file, from_wrapper=False):
        #if from_wrapper:
        out = docker_exec(patp, f"click -b urbit -kp -i {hoon_file} {patp}").output.decode("utf-8").strip().split("\n")
        avow = False
        result = ""
        trace = ""
        for ln in out:
            if (not avow) and ('%avow' not in ln):
                trace = f"{trace}{ln}\n"
            else:
                avow = True
                result = f"{result}{ln}\n"

        return {"trace":trace,"result":result.strip()}

    #
    #   Hoon file builder
    #

    # Create
    def create_hoon(self, payload={}):
        try:
            hoon_file = f"{self.urb._volume_directory}/{self.patp}/_data/{self.name}.hoon"
            with open(hoon_file,'w') as f :
                f.write(self.get_hoon(self.name, payload))
                f.close()
        except Exception:
            Log.log(f"{patp}: Creating {self.name}.hoon failed")
            return False
        return True

    # Delete
    def delete_hoon(self):
        try:
            hoon_file = f"{self.urb._volume_directory}/{self.patp}/_data/{self.name}.hoon"
            if os.path.exists(hoon_file):
                os.remove(hoon_file)
        except Exception as e:
            Log.log(f"{self.patp}: Deleting {self.name}.hoon failed: {e}")
            return False
        return True

    #
    #   Filters
    #

    # +code
    def filter_code(self, data):
        code = "not-yet"
        result = str(data['result'])
        if len(str(result)) > 0:
            res = result.split(' ')[-1][1:-1]
            if len(res) == 27:
                code = res
        else:
            return False
        return code

    # |pack and |meld
    def filter_success(self, data):
        return 'success' in str(data['result'])

    #
    #   Hoon
    #

    def get_hoon(self, name, payload={}):
        # +code
        if name == "code":
# code.hoon
            return """
=/  m  (strand ,vase)
;<    our=@p
    bind:m
  get-our
;<    code=@p
    bind:m
  (scry @p /j/code/(scot %p our))
(pure:m !>((crip (slag 1 (scow %p code)))))
"""
        if name == "s3":
# s3
            return f"""
=/  m  (strand ,vase)
;<    our=@p
    bind:m
  get-our
;<    ~
    bind:m
  (poke [our %storage] %s3-action !>([%set-endpoint '{payload['endpoint']}']))
;<    ~
    bind:m
  (poke [our %storage] %s3-action !>([%set-access-key-id '{payload['acc']}']))
;<    ~
    bind:m
  (poke [our %storage] %s3-action !>([%set-secret-access-key '{payload['secret']}']))
;<    ~
    bind:m
  (poke [our %storage] %s3-action !>([%set-current-bucket '{payload['bucket']}']))
(pure:m !>('success'))
"""
        if name == "s3-legacy":
# s3-legacy
            return f"""
=/  m  (strand ,vase)
;<    our=@p
    bind:m
  get-our
;<    ~
    bind:m
  (poke [our %s3-store] %s3-action !>([%set-endpoint '{payload['endpoint']}']))
;<    ~
    bind:m
  (poke [our %s3-store] %s3-action !>([%set-access-key-id '{payload['acc']}']))
;<    ~
    bind:m
  (poke [our %s3-store] %s3-action !>([%set-secret-access-key '{payload['secret']}']))
;<    ~
    bind:m
  (poke [our %s3-store] %s3-action !>([%set-current-bucket '{payload['bucket']}']))
(pure:m !>('success'))
"""
