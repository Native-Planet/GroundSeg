# Modules
import docker

# GroundSeg modules
from utils import Utils
from log import Log

client = docker.from_env()

class UrbitDocker:

    def start(self, patp, updater_info):
        Log.log(f"{patp}: Attempting to start container")

        # Check if patp is valid
        if not Utils.check_patp(patp):
            Log.log(f"{patp}: Invalid patp")
            return "invalid"

        # Get container
        c = self.get_container(patp)
        if not c:
            return "failed"

        # Get status
        if c.status == "running":
            Log.log(f"{patp}: Container already started")
            return "succeeded"

        # Start ship container
        try:
            c.start()
            Log.log(f"{patp}: Successfully started container")
            return "succeeded"
        except:
            Log.log(f"{patp}: Failed to start container")
            return "failed"

    def stop(self, patp):
        Log.log(f"{patp}: Attempting to stop container")
        c = self.get_container(patp)
        if c:
            try:
                c.stop()
            except Exception as e:
                Log.log(f"{patp}: Failed to stop container")
                return False

        Log.log(f"{patp}: Container stopped")
        return True

    def get_container(self, patp):
        try:
            c = client.containers.get(patp)
            return c
        except:
            Log.log(f"{patp}: Container not found")
            return False


    def create(self, config, updater_info, arch, vol_dir, key=''):
        patp = config['pier_name']
        tag = config['urbit_version']
        if tag == "latest" or tag == "edge":
            sha = f"{arch}_sha256"
            image = f"{updater_info['repo']}:tag@sha256:{updater_info[sha]}"
        else:
            image = f"{updater_info['repo']}:{updater_info['tag']}"

        Log.log(f"{patp}: Attempting to create container")

        if self._pull_image(image, patp):
            v = self._build_volume(patp, vol_dir)
            if v:
                Log.log(f"{patp}: Creating Mount object")
                mount = docker.types.Mount(target = '/urbit/', source=patp)
                if self._build_container(patp, image, mount, config):
                    return self.add_key(key, patp, vol_dir)

    def delete(self, patp):
        if self.delete_container(patp):
            return self.delete_volume(patp)

    def delete_container(self, patp):
        Log.log(f"{patp}: Attempting to delete container")
        c = self.get_container(patp)
        if not c:
            return True
        try:
            c.remove(force=True)
            Log.log(f"{patp}: Container deleted")
            return True
        except Exception as e:
            Log.log(f"{patp}: Failed to delete container: {e}")

        return False

    def delete_volume(self, patp):
        Log.log(f"{patp}: Attempting to delete volume")
        v = self._get_volume(patp)
        if not v:
            return True
        try:
            v.remove(force=True)
            Log.log(f"{patp}: Volume deleted")
            return True
        except Exception as e:
            Log.log(f"{patp}: Error removing volume: {e}")

        return False

    def add_key(self, key, patp, vol_dir):
        if len(key) > 0:
            Log.log(f"{patp}: Attempting to add key")
            try:
                with open(f'{vol_dir}/{patp}/_data/{patp}.key', 'w') as f:
                    f.write(key)
                    f.close()
                return True
            except Exception as e:
                Log.log(f"{patp}: Failed to add key: {e}")

            return False
        return True

    def full_logs(self, patp):
        c = self.get_container(patp)
        if not c:
            return False
        return c.logs()

    def exec(self, patp, command):
        c = self.get_container(patp)
        if c:
            try:
                res = c.exec_run(command)
                return res
            except Exception as e:
                Log.log(f"{patp}: Unable to exec {command}: {e}")

        return False

    def _pull_image(self, image, patp):
        try:
            Log.log(f"{patp}: Pulling {image}")
            client.images.pull(image)
            return True
        except Exception as e:
            Log.log(f"{patp}: Failed to pull {image}: {e}")
            return False

    def _get_volume(self, patp):
        try:
            v = client.volumes.get(patp)
            Log.log(f"{patp}: Volume found")
            return v
        except:
            Log.log(f"{patp}: Volume not found")
            return False


    def _build_volume(self, patp, vol_dir):
        v = self._get_volume(patp)
        if v:
            return v
        else:
            try:
                Log.log(f"{patp}: Attempting to create new volume")
                v = client.volumes.create(name=patp)
                with open(f'{vol_dir}/{patp}/_data/start_urbit.sh', 'w') as f:
                    script = Utils.start_script()
                    f.write(script)
                    f.close()
                Log.log(f"{patp}: Volume created")
                return v

            except Exception as e:
                Log.log(f"{patp}: Failed to create volume: {e}")
                return False


    def _build_container(self, patp, image, mount, config):
        try:
            Log.log(f"{patp}: Building container")
            command = f'bash /urbit/start_urbit.sh --loom={config["loom_size"]}'

            if config["network"] != "none":
                Log.log(f"{patp}: Network is set to wireguard")
                http = f"--http-port={config['wg_http_port']}"
                ames = f"--port={config['wg_ames_port']}"
                command = f"{command} {http} {ames}"

                c = client.containers.create(
                        image = image,
                        command = command, 
                        name = patp,
                        network = f'container:{config["network"]}',
                        mounts = [mount],
                        detach=True)
            else:
                Log.log(f"{patp}: Network is set to local")

                c = client.containers.create(
                        image = image,
                        command = command, 
                        name = patp,
                        ports = {
                            '80/tcp':config['http_port'],
                            '34343/udp':config['ames_port']
                            },
                        mounts = [mount],
                        detach=True)

            if c:
                Log.log(f"{patp}: Successfully built container")
                return True
            else:
                raise Exception("Container wasn't created")

        except Exception as e:
            Log.log(f"{patp}: Failed to build container: {e}")
            return False


    '''
    def remove_urbit(self):
        self.stop()
        self.container.remove()
        self.volume.remove()


    def set_wireguard_network(self, url, http_port, ames_port, s3_port, console_port):
        self.config['wg_url'] = url
        self.config['wg_http_port'] = http_port
        self.config['wg_ames_port'] = ames_port
        self.config['wg_s3_port'] = s3_port
        self.config['wg_console_port'] = console_port

        self.save_config()
        running = False
        
        if self.is_running():
            self.stop()
            running = True

        self.container.remove()
        
        self.build_container()
        if running:
            self.start()

    def update_wireguard_network(self, url, http_port, ames_port, s3_port, console_port):
        changed = False
        if not self.config['wg_url'] == url:
            Log.log_groundseg(f"{self.pier_name}: Wireguard URL changed from {self.config['wg_url']} to {url}")
            changed = True
            self.config['wg_url'] = url

        if not self.config['wg_http_port'] == http_port:
            Log.log_groundseg(f"{self.pier_name}: Wireguard HTTP Port changed from {self.config['wg_http_port']} to {http_port}")
            changed = True
            self.config['wg_http_port'] = http_port

        if not self.config['wg_ames_port'] == ames_port:
            Log.log_groundseg(f"{self.pier_name}: Wireguard Ames Port changed from {self.config['wg_ames_port']} to {ames_port}")
            changed = True
            self.config['wg_ames_port'] = ames_port

        if not self.config['wg_s3_port'] == s3_port:
            Log.log_groundseg(f"{self.pier_name}: Wireguard S3 Port changed from {self.config['wg_s3_port']} to {s3_port}")
            changed = True
            self.config['wg_s3_port'] = s3_port

        if not self.config['wg_console_port'] == console_port:
            Log.log_groundseg(f"{self.pier_name}: Wireguard Console Port changed from {self.config['wg_console_port']} to {console_port}")
            changed = True
            self.config['wg_console_port'] = console_port

        if changed:
            self.save_config()
            if self.config['network'] != "none":
                Log.log_groundseg(f"{self.pier_name}: Rebuilding container")
                running = False
                
                if self.is_running():
                    self.stop()
                    running = True

                self.container.remove()
                
                self.build_container()
                Log.log_groundseg(f"{self.pier_name}: Rebuilding completed")
                if running:
                    Log.log_groundseg(f"{self.pier_name}: Restarting container")
                    self.start()

    def set_network(self, network):
        if self.config['network'] == network:
            return 0

        running = False
        if self.running:
            self.stop()
            running = True
        
        self.container.remove()
        self.config['network'] = network
        self.save_config()

        self.build_container()

        if running:
            self.start()

        return 0


    def toggle_meld_status(self, loopbackAddr):
        self.config['meld_schedule'] = not self.config['meld_schedule']
        self.save_config()
        try:
            now = int(datetime.utcnow().timestamp())
            if self.config['meld_schedule']:
                if int(self.config['meld_next']) <= now:
                    self.send_meld(loopbackAddr)
        except:
            pass
        
        return 200


    def send_meld(self, lens_addr):
        pack_data = dict()
        meld_data = dict()
        pack_source = dict()
        meld_source = dict()
        sink = dict()

        pack_source['dojo'] = "+hood/pack"
        meld_source['dojo'] = "+hood/meld"

        sink['app'] = "hood"

        pack_data['source'] = pack_source
        meld_data['source'] = meld_source

        pack_data['sink'] = sink
        meld_data['sink'] = sink

        with open(f'{self._volume_directory}/{self.pier_name}/_data/pack.json','w') as f :
            json.dump(pack_data, f)

        with open(f'{self._volume_directory}/{self.pier_name}/_data/meld.json','w') as f :
            json.dump(meld_data, f)

        x = self.container.exec_run(f'curl -s -X POST -H "Content-Type: application/json" -d @pack.json {lens_addr}').output.strip()

        if x:
            y = self.container.exec_run(f'curl -s -X POST -H "Content-Type: application/json" -d @meld.json {lens_addr}').output.strip()

            if y:
                now = datetime.utcnow()

                self.config['meld_last'] = str(int(now.timestamp()))

                hour, minute = self.config['meld_time'][0:2], self.config['meld_time'][2:]
                meld_next = int(now.replace(hour=int(hour), minute=int(minute), second=0).timestamp())
                day = 60 * 60 * 24 * self.config['meld_frequency']
                
                self.config['meld_next'] = str(meld_next + day)
                self.save_config()

                os.remove(f'{self._volume_directory}/{self.pier_name}/_data/pack.json')
                os.remove(f'{self._volume_directory}/{self.pier_name}/_data/meld.json')

                return y

    def send_poke(self, command, data, lens_addr):

        f_data = dict()
        source = dict()
        sink = dict()

        source['dojo'] = f"+landscape!s3-store/{command} '{data}'"
        sink['app'] = "s3-store"

        f_data['source'] = source
        f_data['sink'] = sink

        with open(f'{self._volume_directory}/{self.pier_name}/_data/{command}.json','w') as f :
            json.dump(f_data, f)

        x = self.container.exec_run(f'curl -s -X POST -H "Content-Type: application/json" -d @{command}.json {lens_addr}').output.strip()
        os.remove(f'{self._volume_directory}/{self.pier_name}/_data/{command}.json')

        return x

    def set_minio_endpoint(self, endpoint, access_key, secret, bucket, lens_addr):
        self.send_poke('set-endpoint', endpoint, lens_addr)
        self.send_poke('set-access-key-id', access_key, lens_addr)
        self.send_poke('set-secret-access-key', secret, lens_addr)
        self.send_poke('set-current-bucket', bucket, lens_addr)

        return 200

    def unlink_minio_endpoint(self, lens_addr):
        self.send_poke('set-endpoint', '', lens_addr)
        self.send_poke('set-access-key-id', '', lens_addr)
        self.send_poke('set-secret-access-key', '', lens_addr)
        self.send_poke('set-current-bucket', '', lens_addr)

        return 200

    def set_meld_schedule(self, freq, hour, minute):

        current_meld_next = datetime.fromtimestamp(int(self.config['meld_next']))
        time_replaced_meld_next = int(current_meld_next.replace(hour=hour, minute=minute).timestamp())

        day_difference = freq - self.config['meld_frequency']
        day = 60 * 60 * 24 * day_difference

        self.config['meld_next'] = str(day + time_replaced_meld_next)

        if hour < 10:
            hour = '0' + str(hour)
        else:
            hour = str(hour)

        if minute < 10:
            minute = '0' + str(minute)
        else:
            minute = str(minute)

        self.config['meld_time'] = hour + minute
        self.config['meld_frequency'] = int(freq)
        self.save_config()

        return 200

    #def reset_code(self):
    #    return self.container.exec_run('/bin/reset-urbit-code').output.strip()

    
'''
