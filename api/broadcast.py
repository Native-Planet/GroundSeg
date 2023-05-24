import json
import asyncio

from log import Log

class Broadcast:
    def __init__(self, authed, unauth, ws_util):
        self.authed = authed
        self.unauth = unauth
        self.ws_util = ws_util

    async def unauthorized(self):
        while True:
            try:
                clients = self.unauth.copy()
                for s in clients:
                    if s.open:
                        login = self.ws_util.structure['system']['login']
                        setup = self.ws_util.structure['system']['setup']
                        message = {"system": {"login":login,"setup":setup}}
                        await s.send(json.dumps(message))
                    else:
                        self.unauth.pop(s)
            except Exception as e:
                Log.log(f"broadcast:unauthorized Broadcast fail: {e}")

            await asyncio.sleep(0.5)  # Send the message twice a second

    async def authorized(self):
        while True:
            try:
                clients = self.authed.copy()
                for s in clients:
                    if s.open:
                        message = self.ws_util.structure
                        message['system']['login']['access'] = "authorized"
                        await s.send(json.dumps(message))
                    else:
                        self.authed.pop(s)

            except Exception as e:
                Log.log(f"broadcast:authorized Broadcast fail: {e}")

            await asyncio.sleep(0.5)  # Send the message twice a second

    '''
    async def broadcast_unauthorized(self):
        try:
            clients = self.unauthorized_clients.copy()
            for client in clients:
                if client.open:
                    message = {
                            "system":{
                                "login":{
                                    "access": "unauthorized", #authorized #setup-mode #locked
                                    "attempts": self.config_object.login_status['attempts'],
                                    "cooldown": 0
                                    }
                                },
                            "setup": {
                                "status": "done"
                                }
                            }

                    if self.config_object.login_status['locked']:
                        message['system']['access'] = "not-allowed"

                    end = self.config_object.login_status['end']
                    now = datetime.now()
                    if end > now:
                        message['system']['cooldown'] = int((end - now).total_seconds())

                    sid = self.authorized_clients[client]
                    try:
                        forms = self.ws_util.forms.get(sid)
                        if forms != None:
                            forms = {"forms":forms}
                        else:
                            raise Exception()
                    except:
                        forms = {}
                    message = {**forms, **message}
                    await client.send(json.dumps(message))
                else:
                    self.unauthorized_clients.pop(client)
        except Exception as e:
            Log.log(f"websocket_handler:broadcast_unauthorized Broadcast fail: {e}")
                    '''
