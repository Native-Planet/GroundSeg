#!/bin/bash

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=beta_2

# Directory to save the scrips
SAVE_DIR=/opt/nativeplanet/groundseg

# Docker compose file
sudo mkdir -p $SAVE_DIR
wget -O $SAVE_DIR/docker-compose.yml \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/docker-compose.yml

# Pipe for running commands on host
wget -O $SAVE_DIR/opencmd.sh \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/opencmd.sh
chmod +x $SAVE_DIR/opencmd.sh

# Create pipe file
sudo mkfifo $SAVE_DIR/commands

# systemd unit
sudo wget -O /etc/systemd/system/groundseg.service \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg.service
sudo wget -O /etc/systemd/system/gs-pipefile.service \
	https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/gs-pipefile.service

# Load and start
sudo systemctl enable groundseg
sudo systemctl enable gs-pipefile
sudo systemctl daemon-reload 
sudo systemctl restart groundseg
sudo systemctl restart gs-pipefile
