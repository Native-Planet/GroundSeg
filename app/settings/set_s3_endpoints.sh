#! /bin/bash

curl=$(curl -s -X POST -H "Content-Type: application/json" \
-d @data.json \
http://127.0.0.1:12321)

echo $curl
