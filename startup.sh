#! /usr/bin/bash

if [ "$(sudo lsof -i -P -n | grep 5000)" != "" ]
then
echo "UI already running"
exit 0
else
cd ./app/
sudo python3 app.py &

cd ../ui/
npm install
npm run dev -- --open --host

fi

