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

# Install Requirements
sudo pip3 install -r requirements.txt

sudo apt-get install npm wireguard docker nodejs
sudo npm install -g node


# Make build directory
mkdir build
rm -rf build/*

cp -r app/* build/

# Build Svelte UI
cd ui
npm install 
npm run build

cp -r build/* ../build/ui/

cd ../

