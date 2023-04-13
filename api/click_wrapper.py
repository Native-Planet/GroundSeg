from log import Log

class Click:
    def click_exec(patp, docker_exec, hoon):
        out = docker_exec(patp, f"click -b urbit -kp -i {hoon} {patp}").output.decode("utf-8").strip().split("\n")
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


    def filter_code(patp, data):
        code = "not-yet"
        result = str(data['result'])
        if len(str(result)) > 0:
            res = result.split(' ')[-1][1:-1]
            if len(res) == 27:
                code = res
        else:
            return False

        return code
