#!/bin/bash

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=main

mkdir -p /tmp/nativeplanet
wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
chmod +x /tmp/nativeplanet/docker_install.sh
sudo /tmp/nativeplanet/docker_install.sh

systemctl enable docker
systemctl start docker

wget -O /tmp/nativeplanet/groundseg_install.sh \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg_install.sh

chmod +x /tmp/nativeplanet/groundseg_install.sh
sudo /tmp/nativeplanet/groundseg_install.sh

rm -r /tmp/nativeplanet
