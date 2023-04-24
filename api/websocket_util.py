import json

class WSUtil:
    def make_response(aid, success, msg):
        if success:
            res = {"activity":{aid:{"message":msg,"error": 0}}}
        else:
            res = {"activity":{aid:{"message":msg,"error": 1}}}
        return json.dumps(res)
