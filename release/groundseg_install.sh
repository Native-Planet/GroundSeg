#!/bin/bash

# Add mdns to firewalld in PureOS
sudo firewall-cmd --permanent --add-service=mdns # permanent
sudo firewall-cmd --reload

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=main

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

if [[ "$OSTYPE" == "linux-gnu"* ]]; then

  # systemd units
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

elif [[ "$OSTYPE" == "darwin"* ]]; then
  # launchd daemons
  sudo wget -O /Library/LaunchDaemons/io.nativeplanet.groundseg.plist \
	  https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/io.nativeplanet.groundseg.plist
  sudo wget -O /Library/LaunchDaemons/io.nativeplanet.gs-pipefile.plist \
	  https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/io.nativeplanet.gs-pipefile.plist

  # Load and start
  sudo launchctl load ~/Library/LaunchDaemons/io.nativeplanet.groundseg.plist
  sudo launchctl load ~/Library/LaunchDaemons/io.nativeplanet.gs-pipefile.plist

else
  echo "Unsupported Operating System. Please reach out to ~raldeg/nativeplanet for further assistance"
fi
