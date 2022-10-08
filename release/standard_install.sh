mkdir -p /tmp/nativeplanet
wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
chmod +x /tmp/nativeplanet/docker_install.sh
/tmp/nativeplanet/docker_install.sh

DOCKER_CONFIG=${DOCKER_CONFIG:-$HOME/.docker}
mkdir -p $DOCKER_CONFIG/cli-plugins
wget -O $DOCKER_CONFIG/cli-plugins/docker-compose \
	https://github.com/docker/compose/releases/download/v2.11.2/docker-compose-linux-x86_64
chmod +x $DOCKER_CONFIG/cli-plugins/docker-compose

wget -O /tmp/nativeplanet/groundseg_install.sh \
	https://github.com/nallux-dozryl/GroundSeg/main/release/groundseg_install.sh
chmod +x /tmp/nativeplanet/groundseg_install.sh
sudo /tmp/nativeplanet/groundseg_install.sh 
