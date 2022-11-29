# clear everything
sudo docker stop $(sudo docker ps -aq)
sudo docker rm -f $(sudo docker ps -aq)
sudo docker volume prune -f
sudo docker rmi -f $(sudo docker images -q)
sudo rm -r /opt/nativeplanet/groundseg/*

# build binary
python -m nuitka --clang --onefile api/groundseg.py -o groundseg

# move binary
sudo mv groundseg /opt/nativeplanet/groundseg/groundseg

# restart service
sudo chmod +x /opt/nativeplanet/groundseg/groundseg
sudo systemctl restart groundseg
