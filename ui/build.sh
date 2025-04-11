#!/bin/bash
DOCKER_BUILDKIT=0 docker build --build-arg GS_VERSION="v2.4.5" -t web-builder -f builder.Dockerfile .
container_id=$(docker create web-builder)
docker cp $container_id:/webui/build ./web
rm -rf ../goseg/web
mv web ../goseg/