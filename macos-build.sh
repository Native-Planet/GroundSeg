# clear everything
#sudo docker stop $(sudo docker ps -aq)
#sudo docker rm -f $(sudo docker ps -aq)
#sudo docker volume prune -f
#sudo docker rmi -f $(sudo docker images -q)
#sudo rm -r /opt/nativeplanet/groundseg/*

# build binary
python3 -m nuitka --onefile api/groundseg.py -o groundseg

# move binary
sudo cp groundseg /opt/nativeplanet/groundseg/groundseg

# make binary executable
sudo chmod +x /opt/nativeplanet/groundseg/groundseg

# restart service
sudo launchctl stop groundseg
sudo launchctl start groundseg
