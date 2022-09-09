# Copy files and start service
#! /usr/bin/bash

if [[ "$EUID" = 0 ]]; then
    echo "(1) already root"
else
    sudo -k # make sure to ask for password on next sudo
    if sudo true; then
        echo "(2) correct password"
    else
        echo "(3) wrong password"
        exit 1
    fi
fi


sudo mkdir -p /opt/nativeplanet/groundseg/
sudo cp -r build/* /opt/nativeplanet/groundseg
sudo cp groundseg.service /etc/systemd/system/

sudo systemctl stop groundseg
sudo systemctl enable groundseg
sudo systemctl daemon-reload
sudo systemctl start groundseg
