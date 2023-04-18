import json

class WSUtil:
    def make_response(aid, success):
        if success:
            res = {"activity":{aid:{"message":"auth-failed","error": 1}}}
        else:
            res = {"activity":{aid:{"message":"received","error": 0}}}
        return json.dumps(res)
