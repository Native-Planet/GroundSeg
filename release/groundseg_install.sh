#!/bin/bash
sudo mkdir -p /opt/nativeplanet/groundseg/
wget -O /opt/nativeplanet/groundseg/docker-compose.yml \
	https://raw.githubusercontent.com/nallux-dozryl/GroundSeg/main/release/docker-compose.yml

sudo wget -O /etc/systemd/system/groundseg.service \
	https://raw.githubusercontent.com/nallux-dozryl/GroundSeg/main/release/groundseg.service
sudo systemctl enable groundseg
sudo systemctl daemon-reload
sudo systemctl restart groundseg

rm -r /tmp/nativeplanet
