#!/bin/bash

# Add mdns to firewalld in PureOS
sudo firewall-cmd --permanent --add-service=mdns # permanent
sudo firewall-cmd --reload

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=master
TAG=v1.1.7_latest
DEVICE_ARCH=$(uname -m)

# Directory to save the scrips
SAVE_DIR=/opt/nativeplanet/groundseg
sudo mkdir -p $SAVE_DIR

# Stop GroundSeg if running
sudo systemctl stop groundseg

# Download GroundSeg binary
if [[ $DEVICE_ARCH == "aarch64" ]]; then
  sudo wget -O $SAVE_DIR/groundseg https://files.native.computer/bin/groundseg_arm64_$TAG
elif [[ $DEVICE_ARCH == "x86_64" ]]; then
  sudo wget -O $SAVE_DIR/groundseg https://files.native.computer/bin/groundseg_amd64_$TAG
fi

sudo chmod +x $SAVE_DIR/groundseg

if [[ "$OSTYPE" == "linux-gnu"* ]]; then

  # systemd unit
  sudo wget -O /etc/systemd/system/groundseg.service \
	  https://raw.githubusercontent.com/$ACC/$REPO/$BRANCH/release/groundseg.service

  # Load and start
  sudo systemctl enable groundseg
  sudo systemctl daemon-reload 
  sudo systemctl restart groundseg

else
  echo "Unsupported Operating System. Please reach out to ~raldeg/nativeplanet for further assistance"
fi
