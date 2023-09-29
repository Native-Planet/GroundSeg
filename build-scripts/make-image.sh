BUILD_PATH=$(echo $(realpath "$(dirname "$0")"))
docker build -t nativeplanet/groundseg-builder:3.11.2 $BUILD_PATH
docker push nativeplanet/groundseg-builder:3.11.2
