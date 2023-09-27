package accesspoint

import (
	"fmt"
	"goseg/logger"
	"log"
	"strings"
)

func startRouter() bool {
	preStart()
	cmd := "ifconfig " + wlan + " up " + ip + " netmask " + netmask
	logger.Logger.Debug(fmt.Sprintf("created interface: mon.%s on IP: %s", wlan, ip))
	executeShell(cmd)

	logger.Logger.Debug(fmt.Sprintf("wait.."))
	executeShell("sleep 2")

	ipParts := ip[:strings.LastIndex(ip, ".")]

	logger.Logger.Debug(fmt.Sprintf("enabling forward in sysctl."))
	executeShell("sysctl -w net.ipv4.ip_forward=1")

	if inet != "" {
		logger.Logger.Debug(fmt.Sprintf("creating NAT using iptables: %s <-> %s", wlan, inet))
		executeShell("iptables -P FORWARD ACCEPT")
		executeShell("iptables --table nat --delete-chain")
		executeShell("iptables --table nat -F")
		executeShell("iptables --table nat -X")
		executeShell("iptables -t nat -A POSTROUTING -o " + wlan + " -j MASQUERADE")
		executeShell("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT -m state --state RELATED,ESTABLISHED")
		executeShell("iptables -A FORWARD -i " + wlan + " -o " + wlan + " -j ACCEPT")
	}

	executeShell("iptables -A OUTPUT --out-interface " + wlan + " -j ACCEPT")
	executeShell("iptables -A INPUT --in-interface " + wlan + " -j ACCEPT")

	cmd = "dnsmasq --dhcp-authoritative --interface=" + wlan + " --dhcp-range=" + ipParts + ".20," + ipParts + ".100," + netmask + ",4h"
	logger.Logger.Debug(fmt.Sprintf("running dnsmasq"))
	executeShell(cmd)

	cmd = "hostapd -B " + hostapdConfigPath
	logger.Logger.Debug(fmt.Sprintf("running hostapd"))
	logger.Logger.Debug(fmt.Sprintf("wait.."))
	executeShell("sleep 2")
	executeShell(cmd)

	logger.Logger.Debug(fmt.Sprintf("hotspot is running."))
	return true
}

// stopRouter stops the router
func stopRouter() bool {
	// Bring down the interface
	executeShell("ifconfig mon." + wlan + " down")

	// Stop hostapd
	log.Println("stopping hostapd")
	executeShell("pkill hostapd")

	// Stop dnsmasq
	log.Println("stopping dnsmasq")
	executeShell("killall dnsmasq")

	// Disable forwarding in iptables
	log.Println("disabling forward rules in iptables.")
	executeShell("iptables -P FORWARD DROP")

	// Delete iptables rules that were added for wlan traffic
	if wlan != "" {
		executeShell("iptables -D OUTPUT --out-interface " + wlan + " -j ACCEPT")
		executeShell("iptables -D INPUT --in-interface " + wlan + " -j ACCEPT")
	}
	executeShell("iptables --table nat --delete-chain")
	executeShell("iptables --table nat -F")
	executeShell("iptables --table nat -X")

	// Disable forwarding in sysctl
	log.Println("disabling forward in sysctl.")
	executeShell("sysctl -w net.ipv4.ip_forward=0")

	log.Println("hotspot has stopped.")
	return true
}

func preStart() {
	// execute 'killall wpa_supplicant'
	executeShell("killall wpa_supplicant")
	// execute 'nmcli radio wifi off' and check for errors
	result, _ := executeShell("nmcli radio wifi off")
	if strings.Contains(strings.ToLower(result), "error") {
		executeShell("nmcli nm wifi off")
	}
	// execute 'rfkill unblock wlan'
	executeShell("rfkill unblock wlan")
	// execute 'sleep 1'
	executeShell("sleep 1")
}
