
class Setup:
    def handle_anchor(data, config, wg, urbit):
        # set endpoint
        if 'skip' in data:
            config.config['firstBoot'] = False
            config.save_config()
            return 200

        changed = wg.change_url(data['endpoint'], urbit)

        # register key
        if changed == 200:
            endpoint = config.config['endpointUrl']
            api_version = config.config['apiVersion']
            url = f"https://{endpoint}/{api_version}"
            registered = wg.register_device(url, data['key'])

            if registered == 400:
                return 401

            if registered == 200:
                config.config['firstBoot'] = False
                config.save_config()

            return registered

        return 200
