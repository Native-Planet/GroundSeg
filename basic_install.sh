mkdir -p /tmp/nativeplanet
wget -O /tmp/nativeplanet/docker_install.sh https://get.docker.com/
chmod +x /tmp/nativeplanet/docker_install.sh
/tmp/nativeplanet/docker_install.sh
rm /tmp/nativeplanet/docker_install.sh
sudo docker-compose up --build
