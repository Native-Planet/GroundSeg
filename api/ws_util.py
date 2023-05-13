import json
from log import Log

class WSUtil:
    structure = {}
    forms = {}

    def __init__(self, config):
        self.config = config.config

    #
    #   Broadcasts
    #

    # send activity response
    def make_activity(self, aid, success, msg):
        if success:
            res = {"activity":{aid:{"message":msg,"error": 0}}}
        else:
            res = {"activity":{aid:{"message":msg,"error": 1}}}
        return json.dumps(res)

    # Broadcast action for System and Updates
    def system_broadcast(self, category, module, action, info=""):
        try:
            # Whitelist of categories allowed to broadcast
            whitelist = ['system','updates']
            if category not in whitelist:
                raise Exception(f"Error. Category '{category}' not in whitelist")

            # Category
            try:
                if not self.structure.get(category) or not isinstance(self.structure[category], dict):
                    self.structure[category] = {}
            except Exception as e:
                raise Exception(f"failed to set category '{category}': {e}") 

            # Module
            try:
                if not self.structure[category].get(module) or not isinstance(self.structure[category], dict):
                    self.structure[category][module] = {}
            except Exception as e:
                raise Exception(f"failed to set module '{module}': {e}") 

            # Action
            self.structure[category][module][action] = info

        except Exception as e:
            Log.log(f"ws-util:system-broadcast {e}")
            return False

        return True

    # Broadcast action for Urbits
    def urbit_broadcast(self, patp, module, action, info=""):
        try:
            # Set root to structure
            root = self.structure
            # Category
            if not self.structure.get('urbits') or not isinstance(self.structure['urbits'], dict):
                self.structure['urbits'] = {}

            # Set root to urbits
            root = root['urbits']
            # Patp
            if not root.get(patp) or not isinstance(root[patp], dict):
                root[patp] = {}

            # Set root to patp
            root = root[patp]
            # Module
            if not root.get(module) or not isinstance(root[module], dict):
                root[module] = {}

            # Set root to patp
            root = root[module]
            # Action
            root[action] = info

        except Exception as e:
            Log.log(f"ws-util:urbit-broadcast {e}")
            return False
        return True

    #
    #   Form Management
    #

    # modify form
    def edit_form(self, data, template):
        # form belongs to which session
        root = self.forms
        sid = data['sessionid']
        if not root.get(sid) or not isinstance(root[sid], dict):
            root[sid] = {}
        # template
        root = root[sid]
        if not root.get(template) or not isinstance(root[template], dict):
            root[template] = {}

        # key, value
        root = root[template]
        item = data['payload']['item']
        value = data['payload']['value']

        if item == "ships":
            if isinstance(value, str):
                if value == "all":
                    root[item] = self.config['piers'].copy()
                elif value == "none":
                    root[item] = []
            elif not root.get(item) or not isinstance(root[item], list):
                root[item] = value
            else:
                for patp in value:
                    if patp in root[item]:
                        root[item].remove(patp)
                    else:
                        root[item].append(patp)
        else:
            root[item] = value

    # read form
    def grab_form(self, sid, template, item):
        try:
            return self.forms[sid][template][item]
        except:
            return None

    # delete form
    def delete_form(self, sid, template):
        try:
            self.form[sid][template] = {}
            return True
        except:
            return False


    #
    #   Registration  
    #

    # check if service exists for patp
    def services_exist(self, patp, subdomains, is_registered=False):
        # Define services
        services = {
                    "urbit-web":False,
                    "urbit-ames":False,
                    "minio":False,
                    "minio-console":False,
                    "minio-bucket":False
                    }
        for ep in subdomains:
            ep_patp = ep['url'].split('.')[-3]
            ep_svc = ep['svc_type']
            if ep_patp == patp:
                for s in services.keys():
                    if ep_svc == s:
                        if is_registered:
                            services[s] = ep['status']
                        else:
                            services[s] = True
        return services
