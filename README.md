# Native Planet GroundSeg
This is the Control Software for the NP System. 

It expects a Linux Box with `systemd` and `apt` available.

# Single Command Installation
```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/download.sh \
https://raw.githubusercontent.com/nallux-dozryl/GroundSeg/main/download.sh && \
sudo chmod +x /tmp/nativeplanet/download.sh && \
sudo /tmp/nativeplanet/download.sh
```
## Building
To build please run the `build.sh` script in the root directory.

This will create a build folder which will contain the complete webapp

## Installation
To install build the software first and run the `install.sh` script in the root directory.

This will copy the build directory to `/opt/nativeplanet/groundseg/` and the service file to the correct directory

## Development Running
To run this for development use the `startup.sh` script. This will run the python flask app and the UI app as seperate systems.


## TODO 

1. Ability to change the Anchor endpoint. if a user wants to host thier own anchor endpoint and not subscribe we need to let them change the endpoint
2. Refactor the code to be a bit cleaner
3. Add bitcoin node support
4. While Urbit is booting or uploading allow the user to begin setting up MinIO through the UI
5. After Urbit is booted allow user (with the press of a single button) to add S3 support and Bitcoin node support (if installed)
6. Add functionality to check for and update system with the click of a button
7. Add ability to completely turn on/off wifi from settings
8. Setup a cronjob during the install script that run a `|pack` and `|meld` perioidically on each urbit container


## Notes
### How to add s3 endpoints through dojo
```
~sampel-palnet:dojo> :s3-store|set-endpoint 'ams3.digitaloceanspaces.com'
~sampel-palnet:dojo> :s3-store|set-access-key-id 'ACCESSKEY'
~sampel-palnet:dojo> :s3-store|set-secret-access-key '5eCrEtK3Y/8L4H8L4H'
~sampel-palnet:dojo> :s3-store|set-current-bucket 'bucketname'
```

