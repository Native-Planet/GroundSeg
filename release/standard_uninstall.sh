#!/bin/bash

# Remove images
sudo docker rmi -f linuxserver/wireguard \
  quay.io/minio/minio \
  nativeplanet/groundseg_api \
  linuxserver/wireguard \
  quay.io/minio/minio \
  containrrr/watchtower \
  tloncorp/urbit 

# linux
sudo systemctl stop docker
sudo systemctl stop groundseg
sudo systemctl stop gs-pipefile

# macOS
sudo launchctl unload /Library/LaunchDaemons/io.nativeplanet.groundseg.plist
sudo launchctl unload /Library/LaunchDaemons/io.nativeplanet.gs-pipefile.plist

# Remove installed files
sudo rm -r /usr/local/lib/docker
sudo rm -r /opt/nativeplanet/groundseg
sudo rm -r /tmp/nativeplanet

# linux
sudo rm -r /etc/systemd/system/groundseg.service 
sudo rm -r /etc/systemd/system/gs-pipefile.service 

# macOS
sudo rm -r /Library/LaunchDaemons/io.nativeplanet.groundseg.plist
sudo rm -r /Library/LaunchDaemons/io.nativeplanet.gs-pipefile.plist

echo "GroundSeg and Docker uninstalled!"
