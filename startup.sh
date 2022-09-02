#! /usr/bin/bash

cd ./app/
sudo python3 app.py &

cd ../ui/
npm install
npm run dev -- --open --host

