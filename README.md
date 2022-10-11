# Native Planet GroundSeg
This is the Control Software for the NP System. 

Groundseg requires `docker` and `docker-compose` 

## Installation

**Disclaimer:** This software is installed with `sudo` as this is required for Groundseg to work properly.

### Standard Installation (Recommended)
This installs `docker`, `docker-compose` and the required images for GroundSeg to work.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/standard_install.sh \
https://raw.githubusercontent.com/nallux-dozryl/GroundSeg/main/release/standard_install.sh && \
sudo chmod +x /tmp/nativeplanet/standard_install.sh && \
sudo /tmp/nativeplanet/standard_install.sh
```

### Groundseg Only

This downloads and runs the compose file. Only use this if you already have `docker` and `docker-compose` installed.
```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/groundseg_install.sh \
https://raw.githubusercontent.com/nallux-dozryl/GroundSeg/main/release/groundseg_install.sh && \
sudo chmod +x /tmp/nativeplanet/groundseg_install.sh && \
sudo /tmp/nativeplanet/groundseg_install.sh
```

## Updating from Beta-1.0.0 (Assembly 2022 Demo Version)
In order to prevent conflicting configs we will have to copy the config files to the new location.
```
sudo mkdir -p /var/lib/docker/volumes/groundseg_settings/_data && \
sudo cp -r /opt/nativeplanet/groundseg/settings/* /var/lib/docker/volumes/groundseg_settings/_data/
```

**Optional but recommended:** Delete the old files.
```
sudo rm -r /opt/nativeplanet/groundseg/*
```

Lastly, run either one of the install commands.


## Development and Building From Source
1. Clone this repository
2. `export HOST_HOSTNAME=$(hostname)` 
3. Run `sudo docker-compose up --build` in the root directory of the repository.

## TODO 

1. Refactor the code to be a bit cleaner
2. Add bitcoin node support
3. After Urbit is booted allow user (with the press of a single button) to add S3 support and Bitcoin node support (if installed)
4. Setup a cronjob during the install script that run a `|pack` and `|meld` perioidically on each urbit container
