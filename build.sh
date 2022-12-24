#!/bin/bash

GS_PATH=$(echo $(realpath "$(dirname "$0")"))
builder() {
  mkdir -p $GS_PATH/binary
  sudo docker run --rm -v $GS_PATH/api:/api -v $GS_PATH/binary:/binary groundseg-builder:0.1.0
}

standard_build() {
  builder
  echo "Standard build complete. GroundSeg binary in $GS_PATH/binary"
}

clean_docker() {
  echo "Wiping all docker containers, volumes and images."
  sudo docker stop $(sudo docker ps -aq)
  sudo docker rm -f $(sudo docker ps -aq)
  sudo docker volume prune -f
  sudo docker rmi -f $(sudo docker images -q)
  sudo rm -r /opt/nativeplanet/groundseg/*
  echo "Starting build"
  builder
  echo "Clean build complete. GroundSeg binary in $GS_PATH/binary"
}

# Check if --clean is passed
clean_flag=false
for flag in "$@"
do
  if [ "$flag" == "--clean" ]; then
    clean_flag=true
  fi
done

if ! $clean_flag; then
  standard_build
  exit 1
fi

# Loop through all the arguments passed
for arg in "$@"
do
  case $arg in
    "--clean")
      clean_docker
      ;;
    "--prod")
      sudo mkdir -p /opt/nativeplanet/groundseg
      sudo cp $GS_PATH/binary/groundseg /opt/nativeplanet/groundseg/groundseg
      echo "Copied GroundSeg binary to /opt/nativeplanet/groundseg/groundseg"
      ;;
    "--start")
      echo "Start WIP"
      ;;
    *)
      # If an invalid argument is passed, exit the script
      echo "Invalid argument: $arg"
      ;;
  esac
done

# clear everything

# build binary
#python -m nuitka --clang --onefile api/groundseg.py -o groundseg

# move binary
#sudo cp groundseg /opt/nativeplanet/groundseg/groundseg

# restart service
#sudo chmod +x /opt/nativeplanet/groundseg/groundseg
#sudo systemctl restart groundseg
