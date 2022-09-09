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
sudo apt-get install ca-certificates curl gnupg lsb-release
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Get latest Nodejs
curl -sL https://deb.nodesource.com/setup_18.x | sudo -E bash -

# Install Requirements
sudo apt-get -y install wireguard nodejs python3-pip
sudo pip3 install -r requirements.txt


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

