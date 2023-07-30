import nmcli

def get_wifi_device():
    for d in nmcli.device():
        if d.device_type == 'wifi':
            return d.device
    return "none"

def list_wifi_ssids():
    return [x.ssid for x in nmcli.device.wifi() if len(x.ssid) > 0]

def wifi_connect(ssid, pwd):
    try:
        nmcli.device.wifi_connect(ssid, pwd)
        print(f"WiFi: Connected to: {ssid}")
    except Exception as e:
        print(f"WiFi: Failed to connect to network: {e}")

def toggle_wifi():
    try:
        if nmcli.radio.wifi():
            nmcli.radio.wifi_off()
            print(f"Wifi: Turned WiFi Off")
        else:
            nmcli.radio.wifi_on()
            print(f"Wifi: Turned WiFi On")
            return "on"
    except Exception as e:
        print(f"Wifi: Can't toggle WiFi status: {e}")
    return "off"

