import docker
import sys

client = docker.from_env()

class MCDocker:
    _mc_img = "minio/mc:latest"

    def __init__(self):
        client.images.pull(self._mc_img)

        containers = client.containers.list(all=True)

        for c in containers:
            if c.name == 'minio_client':
                try:
                    c.stop()
                    c.remove()
                except Exception as e:
                    print(f"MC removal error: {e}", file=sys.stderr)

        self.container = client.containers.run(
                    image= f'{self._mc_img}',
                    labels = {"com.centurylinklabs.watchtower.enable":"true"},
                    network='container:wireguard',
                    entrypoint='/bin/bash',
                    stdin_open=True,
                    name = 'minio_client',
                    detach=True
                    )

    def mc_setup(self, patp, port, pwd):
        self.container.exec_run(f"mc alias set patp_{patp} http://localhost:{port} {patp} {pwd}")
        self.container.exec_run(f"mc anonymous set public patp_{patp}/bucket")
        return 200

    def make_service_account(self, patp, acc, pwd):
        x = None

        print('Updating service account credentials', file=sys.stderr)
        x = self.container.exec_run(f"mc admin user svcacct edit \
                --secret-key '{pwd}' \
                patp_{patp} {acc}", tty=True).output.decode('utf-8').strip()

        if 'ERROR' in x:
            print('Service account does not exist. Creating new account...', file=sys.stderr)
            x = self.container.exec_run(f"mc admin user svcacct add \
                    --access-key '{acc}' \
                    --secret-key '{pwd}' \
                    patp_{patp} {patp}").output.decode('utf-8').strip()
        
            if 'ERROR' in x:
                return 400

        return 200
        
