#!/bin/bash

# Location of scripts
ACC=nallux-dozryl
REPO=GroundSeg
BRANCH=main

mkdir -p /tmp/nativeplanet
wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
chmod +x /tmp/nativeplanet/docker_install.sh
sudo /tmp/nativeplanet/docker_install.sh

sudo mkdir -p /usr/local/lib/docker/cli-plugins
sudo wget -O /usr/local/lib/docker/cli-plugins/docker-compose \
	https://github.com/docker/compose/releases/download/v2.11.2/docker-compose-linux-x86_64
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

systemctl enable docker
systemctl start docker

wget -O /tmp/nativeplanet/groundseg_install.sh \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg_install.sh

chmod +x /tmp/nativeplanet/groundseg_install.sh
sudo /tmp/nativeplanet/groundseg_install.sh

rm -r /tmp/nativeplanet
