#/bin/bash
docker build -t web-builder -f builder.Dockerfile
docker run --rm -v ../goseg/web:/webui/build web-builder