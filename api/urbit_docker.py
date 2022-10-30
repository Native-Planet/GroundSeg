import docker, json, shutil, threading, time
from datetime import datetime
from minio_docker import MinIODocker

client = docker.from_env()
default_pier_config = {
        "pier_name":"",
        "http_port":8080,
        "ames_port":34343,
        "urbit_version":"latest",
        "minio_version":"latest",
        "minio_password": "",
        "network":"none",
        "wg_url": "nan",
        "wg_http_port": None,
        "wg_ames_port": None,
        "wg_s3_port": None,
        "wg_console_port": None,
        "meld_schedule": False,
        "meld_frequency": 7,
        "meld_time": "0000",
        "meld_last": "0",
        "meld_next": "0"
        }


class UrbitDocker:
    _volume_directory = '/var/lib/docker/volumes'

    def __init__(self,pier_config):
        self.config = pier_config
        client.images.pull(f'tloncorp/urbit:{self.config["urbit_version"]}')
        self.pier_name = self.config['pier_name']
        self.buildUrbit()
        self.running = (self.container.attrs['State']['Status'] == 'running' )
        if(self.is_running()):
            self.stop()
            self.start()

        self.save_config()

    def buildVolume(self):
        volumes = client.volumes.list()
        for v in volumes:
            if self.pier_name == v.name:
                self.volume = v
                shutil.copy('/app/start_urbit.sh', 
                        f'{self._volume_directory}/{self.pier_name}/_data/start_urbit.sh')
                return

        self.volume = client.volumes.create(name=self.pier_name)
        shutil.copy('/app/start_urbit.sh',
                f'{self._volume_directory}/{self.pier_name}/_data/start_urbit.sh')

    def buildContainer(self):
        containers = client.containers.list(all=True)
        for c in containers:
            if(self.pier_name == c.name):
                self.container = c
                return
        if(self.config["network"] != "none"):
            command = f'bash /urbit/start_urbit.sh --http-port={self.config["wg_http_port"]} \
                                          --port={self.config["wg_ames_port"]}'
            self.container = client.containers.create(
                                    f'tloncorp/urbit:{self.config["urbit_version"]}',
                                    command = command, 
                                    name = self.pier_name,
                                    network = f'container:{self.config["network"]}',
                                    mounts = [self.mount],
                                    detach=True)
        else:
            command = f'bash /urbit/start_urbit.sh --http-port={self.config["wg_http_port"]} \
                                          --port={self.config["wg_ames_port"]}'
            self.container = client.containers.create(
                                    f'tloncorp/urbit:{self.config["urbit_version"]}',
                                    ports = {'80/tcp':self.config['http_port'], 
                                             '34343/udp':self.config['ames_port']},
                                    name = self.pier_name,
                                    mounts = [self.mount],
                                    detach = True)


    def set_wireguard_network(self, url, http_port, ames_port, s3_port, console_port):
        self.config['wg_url'] = url
        self.config['wg_http_port'] = http_port
        self.config['wg_ames_port'] = ames_port
        self.config['wg_s3_port'] = s3_port
        self.config['wg_console_port'] = console_port

        self.save_config()
        running = False
        
        if(self.is_running()):
            self.stop()
            running = True

        self.container.remove()
        
        self.buildContainer()
        if(running):
            self.start()

    def set_network(self, network):
        if(self.config['network'] == network):
            return 0

        running = False
        if(self.is_running()):
            self.stop()
            running = True
        
        self.container.remove()
        self.config['network'] = network
        self.save_config()

        self.buildContainer()

        if(running):
            self.start()

        return 0

    def toggle_meld_status(self):
        self.config['meld_schedule'] = not self.config['meld_schedule']
        self.save_config()
        
        return 200

    def save_config(self):
        with open(f'settings/pier/{self.pier_name}.json', 'w') as f:
            json.dump(self.config, f, indent = 4)

    def buildUrbit(self):
        self.buildVolume()
        self.mount = docker.types.Mount(target = '/urbit/', source =self.pier_name)
        self.buildContainer()
    
    def removeUrbit(self):
        self.stop()
        self.container.remove()
        self.volume.remove()

    def add_key(self, key_value):
        with open(f'{self._volume_directory}/{self.pier_name}/_data/{self.pier_name}.key', 'w') as f:
            f.write(key_value)

    def copyFolder(self,folder_loc):
        from distutils.dir_util import copy_tree
        copy_tree(folder_loc,f'{self._volume_directory}/{self.pier_name}/_data/')

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
        print(f"{command} {x.decode('utf-8')}")

        return x

    def set_minio_endpoint(self, endpoint, access_key, secret, bucket, lens_addr):
        self.send_poke('set-endpoint', endpoint, lens_addr)
        self.send_poke('set-access-key-id', access_key, lens_addr)
        self.send_poke('set-secret-access-key', secret, lens_addr)
        self.send_poke('set-current-bucket', bucket, lens_addr)

    def get_code(self, lens_addr):
        f_data = dict()
        source = dict()
        sink = dict()

        source['dojo'] = "+code"
        sink['stdout'] = None

        f_data['source'] = source
        f_data['sink'] = sink

        with open(f'{self._volume_directory}/{self.pier_name}/_data/code.json','w') as f :
            json.dump(f_data, f)

        x = self.container.exec_run(f'curl -s -X POST -H "Content-Type: application/json" -d @code.json {lens_addr}').output.decode('utf-8').strip().split('\\')[0][1:]

        return x

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

    def start(self):
        self.container.start()
        self.running=True
        return 0

    def stop(self):
        self.container.stop()
        self.running=False
        return 0

    def logs(self):
        return self.container.logs()
    
    def is_running(self):
        return self.running
