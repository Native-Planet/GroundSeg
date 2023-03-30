#!/bin/bash

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=master

DIST="$(. /etc/os-release && echo "$ID")"

echo $DIST

sudo mkdir -p /tmp/nativeplanet
sudo apt -y install network-manager avahi-daemon

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
    sudo wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
    sudo chmod +x /tmp/nativeplanet/docker_install.sh
    sudo /tmp/nativeplanet/docker_install.sh
fi

sudo systemctl enable docker
sudo systemctl start docker

sudo wget -O /tmp/nativeplanet/groundseg_install.sh \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg_install.sh

sudo chmod +x /tmp/nativeplanet/groundseg_install.sh
sudo /tmp/nativeplanet/groundseg_install.sh

sudo rm -r /tmp/nativeplanet
