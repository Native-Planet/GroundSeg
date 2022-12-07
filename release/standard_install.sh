#!/bin/bash

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=main

DIST="$(. /etc/os-release && echo "$ID")"

mkdir -p /tmp/nativeplanet

if [[ "$DIST" == "linuxmint" ]]
then
    sudo apt update
    sudo apt -y install apt-transport-https ca-certificates curl software-properties-common
    sudo apt -y remove docker docker-engine docker.io containerd runc
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu jammy stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt update
    sudo apt install docker-ce docker-ce-cli containerd.io
else
    wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
    chmod +x /tmp/nativeplanet/docker_install.sh
    sudo /tmp/nativeplanet/docker_install.sh
fi

systemctl enable docker
systemctl start docker

wget -O /tmp/nativeplanet/groundseg_install.sh \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg_install.sh

chmod +x /tmp/nativeplanet/groundseg_install.sh
sudo /tmp/nativeplanet/groundseg_install.sh

rm -r /tmp/nativeplanet
