# Native Planet GroundSeg
This is the Control Software for the NP System. 

GroundSeg requires `docker` and `docker-compose` to work.

## Updating to Beta-3.0.0 -- Important!
Due to a major refactor in GroundSeg, you will have to run the [GroundSeg Installation command](#groundseg-only) for the software to run properly.

## Installation

**Disclaimer:** This software runs as `root` on your device. This is required for controlling various aspects of the device.

### Docker + GroundSeg (Recommended)
This installs `docker` and the GroundSeg binary.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/standard_install.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/main/release/standard_install.sh && \
sudo chmod +x /tmp/nativeplanet/standard_install.sh && \
sudo /tmp/nativeplanet/standard_install.sh
```

### Groundseg Only

This downloads and runs the compose file. Only use this if you already have `docker` and `docker-compose` installed.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/groundseg_install.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/main/release/groundseg_install.sh && \
sudo chmod +x /tmp/nativeplanet/groundseg_install.sh && \
sudo /tmp/nativeplanet/groundseg_install.sh
```

## Edge Branch Installation (Unstable)
1. Modify `"updateUrl"` in `/opt/nativeplanet/groundseg/settings/system.json` to `https://version.infra.native.computer/version_edge.csv`
2. `sudo systemctl restart groundseg`


## Development and Building From Source
1. Clone this repository
2. `export HOST_HOSTNAME=$(hostname)` 
3. Run `sudo -E docker-compose up --build` in the root directory of the repository.

## Removing GroundSeg (Uninstall)

### Standard Removal (Recommended)
This removes `docker`, `docker-compose`, GroundSeg related docker containers and images, and the GroundSeg system files.
This **DOES NOT** remove the docker volumes on the device.
```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/standard_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/main/release/standard_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/standard_uninstall.sh && \
sudo /tmp/nativeplanet/standard_uninstall.sh
```

### Groundseg Only

This removes GroundSeg related docker containers and images, and the GroundSeg system files.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/groundseg_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/main/release/groundseg_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/groundseg_uninstall.sh && \
sudo /tmp/nativeplanet/groundseg_uninstall.sh
```

### Uninstall and clear data
This removes `docker`, `docker-compose`, **ALL** docker images, containers and volumes, and the GroundSeg system files.
This wipes all docker and GroundSeg data. Make sure you have exported the data you want saved.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/complete_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/main/release/complete_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/complete_uninstall.sh && \
sudo /tmp/nativeplanet/complete_uninstall.sh
```

## TODO 

1. Add bitcoin node support
2. Onboarding screen
