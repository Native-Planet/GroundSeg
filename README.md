# Native Planet GroundSeg
This is the Control Software for the NP System. 

GroundSeg requires `docker` and `glibc ^2.34` to work.

## Installation

**Disclaimer:** This software runs as `root` on your device. This is required for controlling various aspects of the device.

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

- Make sure you are running Windows 10 version 2004 or higher.

- Open the Command Prompt App from the Start Menu.

- Run the `wsl.exe --install` command.

- Reboot your machine.

- You will be prompted for a user name and password from the Command Prompt.

- Open the Ubuntu app from the Start Menu.

- Run the `sudo apt-get update && sudo apt-get upgrade -y` command.

- Once the installation is complete, run `sudo wget -O - get.groundseg.app | bash`.

## Edge Branch Installation (Unstable)

1. Modify `"updateUrl"` in `/opt/nativeplanet/groundseg/settings/system.json` to `https://version.infra.native.computer/version_edge.csv`
2. `sudo systemctl restart groundseg`

## Development and Building From Source

Coming Soon

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
