import os
import subprocess
from time import sleep

class Swap:
    # Configure Swap
    def configure(self, file, val):
        if val > 0:
            if not os.path.isfile(file):
                self.make_swap(file, val)

            self.start_swap(file)
            swap = self.active_swap(file)

            if swap != val:
                if self.stop_swap(file):
                    print(f"config:swap:configure Removing {file}")
                os.remove(file)

                if self.make_swap(file, val):
                    self.start_swap(file)

            return True


    def start_swap(self, loc):
        try:
            subprocess.call(["swapon", loc])
        except Exception as e:
            print(f"config:swap:start_swap Failed to run swapon: {e}")
            return False
        return True

    def stop_swap(self, loc):
        try:
            subprocess.call(["swapoff", loc])
        except Exception as e:
            print(f"config:swap:stop_swap Failed to run swapoff: {e}")
            return False
        return True

    def make_swap(self, loc, val):
        try:
            subprocess.call(["fallocate", "-l", f"{val}G", loc])
            subprocess.call(["chmod", "600", loc])
            subprocess.call(["mkswap", loc])
        except Exception as e:
            print(f"config:swap:make_swap Failed to make swap: {e}")
            return False
        return True

    def active_swap(self, loc):
        count = 0
        while count < 3:
            try:
                res = subprocess.run(["swapon", "--show"], capture_output=True)
                swap_arr = [x for x in res.stdout.decode("utf-8").split('\n') if loc in x]
                value = [x for x in swap_arr[0].split(" ") if x != ""][2]
                num = int("".join(filter(str.isdigit, value)))
                if 'M' in value:
                    return num / 1024
                return num
            except Exception as e:
                count += 1
                sleep(count * 2)
                return 0

    def max_swap(self, loc, val):
        cap = 32 # arbitrary cap for the webui
        free = cap
        try:
            free = math.ceil(psutil.disk_usage(loc).free / (1024 ** 3)) - 2
            if free > cap:
                free = cap
        except Exception as e:
            if val > 0:
                print(f"config:swap:max_swap Failed to get maximum swap: {e}")
        return free
