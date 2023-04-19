from utils import Utils

class Setup:
    def handle_regions(data, config, wg):
        res = {"error":1,"regions":None}
        if 'endpoint' in data:
            endpoint = data['endpoint']
            api_version = config['apiVersion']
            url = f"https://{endpoint}/{api_version}"
            if wg.get_regions(url, tries=1):
                res['regions'] = Utils.convert_region_data(wg.region_data)
                res['error'] = 0
        return res

    def handle_anchor(data, config, wg, urbit, minio):
        # set endpoint
        if 'skip' in data:
            config.config['firstBoot'] = False
            config.save_config()
            return 200

        changed = wg.change_url(data['endpoint'], urbit, minio)

        # register key
        if changed == 200:
            endpoint = config.config['endpointUrl']
            api_version = config.config['apiVersion']
            url = f"https://{endpoint}/{api_version}"
            if wg.build_anchor(url, data['key'], data['region']):
                minio.start_mc()
                config.config['wgRegistered'] = True
                config.config['wgOn'] = True

                for patp in config.config['piers']:
                    urbit.register_urbit(patp, url)

                config.config['firstBoot'] = False
                if config.save_config():
                    return 200

        return 400

    def handle_password(data, config):
        matching = False
        if data['pubkey'] == Utils.convert_pub(config.login_keys['cur']['pub']):
            matching = "cur"
        elif data['pubkey'] == Utils.convert_pub(config.login_keys['old']['pub']):
            matching = "old"

        if matching:
            decrypted = Utils.decrypt_password(config.login_keys[matching]['priv'],data['password'])
            if config.create_password(decrypted):
                return 200

        return 400
