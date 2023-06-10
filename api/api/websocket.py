import json
import asyncio
from threading import Thread

from auth.auth import Auth

class WSGroundSeg:
    def __init__(self,state):
        self.state = state

    async def handle(self, websocket):
        print("app:handle New Websocket Connection")
        try:
            async for message in websocket:
                ready = self.state.get('ready')
                config_ready = ready.get('config')
                orchestrator_ready = ready.get('orchestrator')

                action = json.loads(message)
                status_code = 1
                msg = "CONFIG_NOT_READY"
                token = None

                if config_ready:
                    # Verify the action
                    status_code, msg, token = Auth(self.state).verify_session(action, websocket)

                    # process action in orchestrator
                    if status_code == 0:
                        if orchestrator_ready:
                            status_code, msg, token = self.handle_request(
                                    action,
                                    websocket,
                                    status_code,
                                    msg,
                                    token
                                    )
                        else:
                            status_code = 1
                            msg = "ORCHESTRATOR_NOT_READY"

                # make and send activity
                activity = self.make_activity(action.get('id'), status_code, msg, token)
                await websocket.send(activity)

        except Exception as e:
            print(f"app:handle Error {e}")

    # receive action
    def handle_request(self, action, websocket, status_code, msg, token):
        print(f"app:handle_request id: {action['id']}")
        try:
            # Get the action category
            cat = action.get('payload').get('category')

            # Does nothing
            if cat == "token":
                pass

            # System
            elif cat == "system":
                status_code, msg, token = self.system_action(action, websocket, status_code, msg)

            elif cat == 'urbits':
                status_code, msg = self.urbits_action(action, websocket, status_code, msg)

                '''
            elif cat == 'updates':
                status_code, msg = self.ws_command_updates(payload)
                '''

            elif cat == 'forms':
                token = None
                if websocket in self.state['clients']['authorized']:
                    status_code, msg = self.forms_action(action, status_code, msg)
                else:
                    status_code = 1
                    msg = "UNAUTHORIZED"
            else:
                status_code = 1
                msg = "INVALID_CATEGORY"
                raise Exception(f"'{cat}' is not a valid category")
        except Exception as e:
            print(f"app:handle_request Error {e}")

        return status_code, msg, token


    def urbits_action(self, data, websocket, status_code, msg):
        token = None
        whitelist = [
                'meld',
                'minio',
                'container',
                'access'
                ]
        payload = data.get('payload')
        id = data.get('id')
        patp = payload.get('patp')
        module = payload.get('module')
        action = payload.get('action')

        if module not in whitelist:
            raise Exception(f"{module} is not a valid module")
        if module not in whitelist:
            status_code = 1
            msg = "INVALID_MODULE"
        else:
            if websocket in self.state['clients']['authorized']:
                # Access
                if module == "access":
                    if action == "toggle":
                        Thread(target=self.orchestrator.ws_urbits.access_toggle, args=(patp,)).start()
                # Pack and Meld
                if module == "meld":
                    if action == "urth":
                        Thread(target=self.orchestrator.ws_urbits.meld_urth,
                               args=(patp,)
                               ).start()
                '''
                # MinIO
                if module == "minio":
                    if action == "link":
                        Thread(target=self.minio_link, args=(patp,)).start()
                    if action == "unlink":
                        Thread(target=self.minio_unlink, args=(patp,)).start()
                # Urbit Docker Container
                if module == "container":
                    if action == "rebuild":
                        Thread(target=self.ws_urbits.container_rebuild,
                               args=(patp,)
                               ).start()
                '''
                return status_code, msg

    # System
    def system_action(self, data, websocket, status_code, msg):
        # hardcoded list of allowed modules
        token = None
        whitelist = [
                'login',
                'startram',
                ]
        payload = data.get('payload')
        id = data.get('id')
        module = payload.get('module')
        action = payload.get('action')

        if module not in whitelist:
            status_code = 1
            msg = "INVALID_MODULE"
        else:
            if module == "login":
                status_code, msg, token = Auth(self.state).handle_login(data,websocket,status_code,msg)
            elif websocket in self.state['clients']['authorized']:
                if module == "startram":
                    self.orchestrator = self.state['orchestrator']
                    if action == "register":
                        Thread(target=self.orchestrator.startram_register, args=(id,)).start()
                    if action == "stop":
                        Thread(target=self.orchestrator.startram_stop).start()
                    if action == "start":
                        Thread(target=self.orchestrator.startram_start).start()
                    if action == "restart":
                        Thread(target=self.orchestrator.startram_restart).start()
                    if action == "endpoint":
                        Thread(target=self.orchestrator.startram_change_endpoint,
                               args=(data['sessionid'],)
                               ).start()
                    if action == "cancel":
                        Thread(target=self.orchestrator.startram_cancel,
                               args=(data['sessionid'],)
                               ).start()

        return status_code, msg, token

    # Forms
    def forms_action(self, action, status_code, msg):
        template = action.get('payload').get('template')
        if template == "startram":
            Thread(target=self.state['orchestrator'].edit_form, args=(action,), daemon=True).start()
        else:
            status_code = 1
            msg = "INVALID_TEMPLATE"
        return status_code, msg

        '''
        try:
            # hardcoded whitelist
            whitelist = [
                    'startram'
                    ]

            payload = data['payload']
            sid = data['sessionid']
            template = payload['template']

            if template in whitelist:
                if template == "startram":
                    self.ws_util.edit_form(data, template)

        except Exception as e:
            raise Exception(e)
        return "succeeded"
        '''

    def make_activity(self, id, status_code, msg, token=None):
        res = {"activity":{id:{"message":msg,"status_code":status_code}}}
        if token:
            res['activity'][id]['token'] = token
        return json.dumps(res)
