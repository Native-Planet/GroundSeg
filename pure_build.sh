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

# Install Docker
sudo apt-get update
sudo apt-get -y install ca-certificates curl gnupg lsb-release docker.io
sudo apt-get install avahi-daemon avahi-discover avahi-utils libnss-mdns mdns-scan


# Get latest Nodejs
curl -sL https://deb.nodesource.com/setup_18.x | sudo -E bash -

# Install Requirements
sudo apt-get -y install wireguard nodejs python3-pip
sudo pip3 install -r requirements.txt


# Make build directory
mkdir build
rm -rf build/*

cp -r app/* build/

# Install mc
curl https://dl.min.io/client/mc/release/linux-amd64/mc -o build/mc

# Build Svelte UI
cd ui
npm install 
npm run build

cp -r * ../build/ui/

cd ../

