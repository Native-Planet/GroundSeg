# Native Planet GroundSeg
This is the Control Software for the NP System. 

GroundSeg requires `docker` and `glibc ^2.31` to work.

GroundSeg works better with StarTram. Read more about [StarTram](https://www.nativeplanet.io/startram)

## Installation

**Disclaimer:** GroundSeg runs with `sudo` privileges on your device. This is required for controlling various aspects of the device.

### Docker + GroundSeg (Recommended)
This installs `docker` and the GroundSeg binary. Use this if you do not know what you're doing.

```
sudo wget -O - get.groundseg.app | bash
```

### Groundseg Only

This downloads the appropriate service file for you init system and the groundseg binary. Docker has to already be installed.

```
sudo wget -O - only.groundseg.app | bash
```

### For Windows

Coming soon

### For MacOS

Coming soon

## Edge Branch Installation (Unstable)

1. Go to `/opt/nativeplanet/groundseg/settings/system.json`
2. Set `"updateBranch" to "edge"
3. `sudo systemctl restart groundseg`

## Building From Source

1. Have docker installed
2. run `build.sh`

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
