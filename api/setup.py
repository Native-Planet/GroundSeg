
class Setup:
    def handle_anchor(data, config):
        # set endpoint
        if 'skip' in data:
            config.config['firstBoot'] = False
            config.save_config()
            return 200

        '''
        changed = self.change_wireguard_url(data['endpoint'])

        # register key
        if changed == 200:
            registered = self.register_device(data['key'])

            if registered == 400:
                return 401

            if registered == 200:
                self.config['firstBoot'] = False
                self.save_config()

            return registered
            '''

        return 200
