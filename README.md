# Native Planet GroundSeg

![Groundseg 2 demo](https://user-images.githubusercontent.com/16911914/271397025-f534f6e3-5c62-4b9a-8cb9-9f1d07e5c29a.gif)

GroundSeg is a software tool that helps users manage and access their multiple Urbit instances. 
It simplifies the process of getting onto the Urbit network and provides a range of additional services 
that enhance the functionality of the user's ship. With a [StarTram](https://www.nativeplanet.io/startram) 
subscription, users can also access their Urbit ship remotely.

## Dependencies

- docker
- systemd

## Installation

**Disclaimer:** GroundSeg runs with `sudo` privileges on your device. This is required for controlling various aspects of the device. For this reason, we recommend a dedicated device.

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

### Switching to the `edge` release channel (Unstable)

1. In `/opt/nativeplanet/groundseg/settings/system.json`, set `"updateBranch"` to `"edge"`
2. `sudo systemctl restart groundseg`

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
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/standard_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/standard_uninstall.sh && \
sudo /tmp/nativeplanet/standard_uninstall.sh
```

### Groundseg Only

This removes GroundSeg related docker containers and images, and the GroundSeg system files.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/groundseg_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/groundseg_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/groundseg_uninstall.sh && \
sudo /tmp/nativeplanet/groundseg_uninstall.sh
```

### Uninstall and clear data
This removes `docker`, `docker-compose`, **ALL** docker images, containers and volumes, and the GroundSeg system files.
This wipes all docker and GroundSeg data. Make sure you have exported the data you want saved.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/complete_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/complete_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/complete_uninstall.sh && \
sudo /tmp/nativeplanet/complete_uninstall.sh
```
