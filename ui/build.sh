#!/bin/bash
DOCKER_BUILDKIT=0 docker build -t web-builder -f builder.Dockerfile .
container_id=$(docker create web-builder)
docker cp $container_id:/webui/build ./web
docker rm $container_id
cp -r ./web/* ../goseg/web