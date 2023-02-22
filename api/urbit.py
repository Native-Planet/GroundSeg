from log import Log
from urbit_docker import UrbitDocker

class Urbit:

    def __init__(self, config):
        self.config_object = config
        self.config = config.config
        self.urb_docker = UrbitDocker()

        # Check if updater information is ready
        branch = self.config['updateBranch']
        count = 0
        while not self.config_object.update_avail:
            count += 1
            if count >= 30:
                break

            Log.log("Urbit: Updater information not yet ready. Checking in 3 seconds")
            sleep(3)

        # Updater Wireguard information
        if self.config_object.update_avail:
            self.updater_info = self.config_object.update_payload['groundseg'][branch]['vere']

        self.start_all(self.config['piers'])


#
#   Urbit Docker commands
#


    # Start all valid containers
    def start_all(self, patps):
        Log.log("Urbit: Starting all ships")
        res = {"failed":[],"succeeded":[],"ignored":[],"invalid":[]}
        if len(patps) < 1:
            Log.log(f"Urbit: No ships detected in system.json! Skipping..")
            return True

        for p in patps:
            status = self.start(p)
            try:
                res[status].append(p)
            except Exception as e:
                Log.log(f"{p}: {e}")

        Log.log(f"Urbit: Start succeeded {res['succeedded']}")
        Log.log(f"Urbit: Start ignored {res['ignored']}")
        Log.log(f"Urbit: Start failed {res['failed']}")
        Log.log(f"Urbit: Patp invalid {res['invalid']}")

        return True

    # Start container
    def start(self, patp):
        file = f"{self.config_object.base_path}/settings/pier/{patp}.json"
        return self.urb_docker.start(patp, self.updater_info, file)

    '''
    # Stop container
    def stop(self):
        return self.wg_docker.stop(self.data)

    def remove(self):
        return self.wg_docker.remove_wireguard(self.data['wireguard_name'])

    # Is container running
    def is_running(self):
        return self.wg_docker.is_running(self.data['wireguard_name'])

    # Container logs
    def logs(self):
        return self.wg_docker.logs(self.data['wireguard_name'])
'''

#
#   API responses
#
    
    def list_ships(self):
        urbits = []
        try:
            for patp in self.config['piers']:
                try:
                    u = dict()
                    c = self.urb_docker.get_container(patp)
                    if c:
                        u['name'] = patp
                        u['running'] = c.status == "running"
                        # TODO: Urbit Config
                        #u['url'] = f'http://{socket.gethostname()}.local:{urbit.config["http_port"]}'

                    #if(urbit.config['network']=='wireguard'):
                    #    u['url'] = f"https://{urbit.config['wg_url']}"

                        urbits.append(u)
                except:
                    Log.log("{patp}: {e}")
        except:
            pass

        return {'urbits': urbits}
