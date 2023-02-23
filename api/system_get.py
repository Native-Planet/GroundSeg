# Python
import subprocess

# Modules
import docker
import nmcli

# GroundSeg modules
from log import Log

class SysGet:
    def get_containers():
        containers = []
        try:
            for c in docker.from_env().containers.list(all=True):
                containers.append(c.name)
        except Exception as e:
            Log.log(f"System: Get container list failed: {e}")

        return containers

    def get_ethernet_status():
        try:
            return not nmcli.radio.wifi()
        except Exception as e:
            Log.log(f"System: Can't get ethernet status: {e}")
            return True

    # Check if wifi is connected
    def get_connection_status():
        try:
            conns = nmcli.connection()
            for con in conns:
                if con.conn_type == "wifi":
                    return con.name
        except:
            Log.log(f"System: Can't get WiFi connection status")

        return ''
