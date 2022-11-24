sudo docker stop $(sudo docker ps -aq)
sudo docker rm -f $(sudo docker ps -aq)
sudo docker volume prune -f
sudo docker rmi -f $(sudo docker images -q)
sudo rm -r /opt/nativeplanet/groundseg/settings
sudo rm -r /opt/nativeplanet/groundseg/uploaded
python -m nuitka --onefile api/groundseg.py
