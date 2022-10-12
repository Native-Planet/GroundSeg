#!/bin/bash

# Location of scripts
ACC=nallux-dozryl
REPO=GroundSeg
BRANCH=main

# Directory to save the scrips
SAVE_DIR=/opt/nativeplanet/groundseg

sudo mkdir -p $SAVE_DIR/commands
wget -O $SAVE_DIR/docker-compose.yml \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/docker-compose.yml

sudo wget -O /etc/systemd/system/groundseg.service \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg.service
sudo systemctl enable groundseg
sudo systemctl daemon-reload
sudo systemctl restart groundseg
