# Python
import sys
import asyncio

# Configs
from config.config import Config

# Updater
from updater.linux import LinUpdate
from updater.binary import BinUpdate

# System
from system.monitor import SysMonitor

# GroundSeg
from groundseg.groundseg import GroundSeg

# APIs
from api.websocket_api import WS
from api.lick_ipc import Lick

async def main():
    # Check Dev Mode
    try:
        dev = sys.argv[1] == "dev"
    except:
        pass
    # Load Config
    base_path = "/opt/nativeplanet/groundseg"
    cfg = Config(base_path,dev)

    # Update Binary
    binary = BinUpdate(cfg,dev)

    # Update Linux
    linux = LinUpdate(cfg, dev)

    # Monitor System
    mon = SysMonitor(cfg, dev)

    # Start GroundSeg
    groundseg = GroundSeg(cfg, dev)

    # APIs
    host = '0.0.0.0'
    port = 8000
    ws = WS(groundseg, host, port, dev)
    ur = Lick(groundseg, dev)

    # Run tasks
    await asyncio.gather(
            cfg.net_check(),
            binary.run(),
            linux.run(),
            mon.ram(),
            mon.cpu(),
            mon.temp(),
            mon.disk(),
            groundseg.loader(),
            groundseg.startram(),
            groundseg.melder(),
            groundseg.wg_refresher(),
            groundseg.docker_updater(),
            ws.run(),
            ur.run()
            )

# Start
asyncio.run(main())
