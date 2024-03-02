#!/bin/bash

# Add mdns to firewalld in PureOS
sudo firewall-cmd --permanent --add-service=mdns # permanent
sudo firewall-cmd --reload

# Location of scripts
ACC=Native-Planet
REPO=GroundSeg
BRANCH=master
TAG=v2.0.15
DEVICE_ARCH=$(uname -m)

# Directory to save the scrips
SAVE_DIR=/opt/nativeplanet/groundseg
sudo mkdir -p $SAVE_DIR

# Stop GroundSeg if running
sudo systemctl stop groundseg

# Raspberry Pi
DEVICE_MODEL=$(grep -i "Model" /proc/cpuinfo)
DOCKER_DIR=/var/lib/docker
if echo "$DEVICE_MODEL" | grep -iq "Raspberry Pi"; then
    echo "                                                  "
    echo "##################################################"
    echo "                                                  "
    echo "            Raspberry Pi detected!                "
    echo "                                                  "
    echo "##################################################"
    echo "                                                  "
    echo "Using an SD card to store Docker data can cause   "
    echo "performance issues and degrade the lifespan of the"
    echo "SD card.                                          "
    echo "                                                  "
    echo "We recommend you to select an alternative location"
    echo "to store the Docker volumes.                      "
    echo "                                                  "
    echo "Example: /mounted/ssd/location/docker             "
    echo "                                                  "
    echo "##################################################"

    read -p "Choose a new directory. Leave blank for default directory (/var/lib/docker): " new_dir

    # Check if user provided new directory
    if [ -n "$new_dir" ]; then
      # Check if provided directory is the same as default
      if [ "$DOCKER_DIR" != "$new_dir" ]; then
        # Check if directory already exists, if no, create it
        if [ ! -d "$new_dir" ]; then
          echo "$new_dir not found!"
          echo "Creating directory: $new_dir"
          sudo mkdir -p $new_dir
        fi

        # Stop Docker
        echo "Stopping Docker"
        sudo systemctl stop docker docker.socket

        # Move Docker directory contents to new location
        echo "Moving Docker from $DOCKER_DIR to $new_dir" \
          && sudo mv $DOCKER_DIR/* $new_dir \
          && echo "Removing old volumes directory: $DOCKER_DIR" \
          && sudo rm -r $DOCKER_DIR \
          && echo "Creating /etc/docker/daemon.json" \
          && sudo echo "{\"data-root\": \"$new_dir\"}" > /etc/docker/daemon.json \
          && sudo systemctl start docker
      else
        echo "Using default volumes directory!"
      fi
    else
      echo "Using default volumes directory!"
    fi
fi

# Download GroundSeg binary
if [[ $DEVICE_ARCH == "aarch64" ]]; then
sudo wget -O $SAVE_DIR/groundseg \
  https://bin.infra.native.computer/groundseg_arm64_${TAG}_latest
elif [[ $DEVICE_ARCH == "x86_64" ]]; then
sudo wget -O $SAVE_DIR/groundseg \
  https://bin.infra.native.computer/groundseg_amd64_${TAG}_latest
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

  echo "##################################################"
  echo "                                                  "
  echo "  Access GroundSeg at:                            "
  echo "   http://$(cat /proc/sys/kernel/hostname).local  "
  echo "                                                  "
  echo "##################################################"

else
  echo "Unsupported Operating System. Please reach out to ~raldeg/nativeplanet for further assistance"
fi
