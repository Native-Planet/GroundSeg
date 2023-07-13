# Python
import sys
import asyncio

# Configs
from config.config import Config

# Updater
from updater.version import VersionServer
from updater.linux import LinUpdate
from updater.binary import BinUpdate

# System
from system.monitor import SysMonitor

# GroundSeg
from groundseg.groundseg import GroundSeg

# APIs
from api.websocket_api import WS
from api.lick_ipc import Lick

# Start Here
async def main():
    # We check if the dev argument is given, then
    # print the appropriate announcment
    try:
        dev = sys.argv[1] == "dev"
        if dev:
            print("---------- Starting GroundSeg in debug mode ----------")
        else:
            raise Exception()
    except:
        print("----------------- Starting GroundSeg -----------------")
    print("------------------ Urbit is love <3 ------------------")

    # Next, we initialize all the relevant classes
    #
    # Config         - handles config files and some system related state
    #
    # Version Server - in charge of getting updated information
    #                  from the specified version server
    # Binary Updater - Checks if the groundseg binary needs updating
    #
    # Linux Updater  - Checks if the underlying linux installation
    #                  needs updating
    #
    # System Monitor - Gets latest system vitals
    #
    # GroundSeg      - The main class for processing requests and interacting
    #                  with docker
    #
    # WS             - The websocket API
    #
    # Lick           - Martian API

    # Load Config
    base_path = "/opt/nativeplanet/groundseg"
    cfg = Config(base_path,dev)

    # Version Server
    version_server = VersionServer(cfg,dev)

    # Binary Updater
    binary = BinUpdate(cfg,base_path,dev)

    # Linux Updater
    linux = LinUpdate(cfg,dev)

    # System Monitor
    mon = SysMonitor(cfg,dev)

    # Start GroundSeg
    groundseg = GroundSeg(cfg,dev)

    # Websocket API
    host = '0.0.0.0'
    port = 8000
    ws = WS(cfg, groundseg, host, port, dev)

    # Lick API
    ur = Lick(groundseg, dev)

    # Now, we run all the services asynchronusly
    # They're all infinite loops except for those
    # specified
    await asyncio.gather(
            cfg.net_check(), # not loop
            version_server.check(),
            binary.run(),
            linux.run(),
            mon.ram(),
            mon.cpu(),
            mon.temp(),
            mon.disk(),
            groundseg.loader(), # not loop, initializes the docker classes
            groundseg.startram(),
            groundseg.melder(),
            groundseg.wg_refresher(),
            groundseg.docker_updater(),
            ws.run(),
            ws.broadcast(),
            ur.run()
            )

# Start
asyncio.run(main())
